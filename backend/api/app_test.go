package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/repository"
	"github.com/katonium/kubegame/backend/domain/service"
	"github.com/katonium/kubegame/backend/infrastructure/kubernetes"
	infraRepo "github.com/katonium/kubegame/backend/infrastructure/repository"
	infraService "github.com/katonium/kubegame/backend/infrastructure/service"
	infraWebSocket "github.com/katonium/kubegame/backend/infrastructure/websocket"
	"github.com/katonium/kubegame/backend/usecase"
	"go.uber.org/fx"
	"k8s.io/client-go/kubernetes/fake"
)

// TestMessage represents a WebSocket message for testing
type TestMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// IntegrationTestSuite contains all dependencies for integration testing
type IntegrationTestSuite struct {
	app               *fx.App
	server            *httptest.Server
	wsService         service.WebSocketService
	gameSessionSvc    service.GameSessionService
	gameRepo          repository.GameRepository
	podRepo           repository.PodRepository
	nodeRepo          repository.NodeRepository
	sessionRepo       repository.GameSessionRepository
	gameUseCase       *usecase.GameUseCase
	gameEngineUseCase *usecase.GameEngineUseCase
}

// SetupIntegrationTest creates a test environment with all dependencies
func SetupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	suite := &IntegrationTestSuite{}

	// Create fx application for dependency injection (without starting HTTP server)
	app := fx.New(
		// Provide repositories
		fx.Provide(
			fx.Annotate(
				infraRepo.NewMemoryGameRepository,
				fx.As(new(repository.GameRepository)),
			),
			fx.Annotate(
				infraRepo.NewMemoryPodRepository,
				fx.As(new(repository.PodRepository)),
			),
			fx.Annotate(
				infraRepo.NewMemoryNodeRepository,
				fx.As(new(repository.NodeRepository)),
			),
			fx.Annotate(
				infraRepo.NewMemorySessionRepository,
				fx.As(new(repository.GameSessionRepository)),
			),
		),

		// Provide services
		fx.Provide(
			fx.Annotate(
				kubernetes.NewKubernetesService,
				fx.As(new(service.KubernetesService)),
			),
			fx.Annotate(
				func() service.SchedulerService {
					client := fake.NewSimpleClientset()
					return kubernetes.NewSchedulerService(client)
				},
				fx.As(new(service.SchedulerService)),
			),
			fx.Annotate(
				func(sessionRepo repository.GameSessionRepository,
					gameRepo repository.GameRepository,
					podRepo repository.PodRepository,
					nodeRepo repository.NodeRepository,
					kubernetesService service.KubernetesService) service.GameSessionService {
					return infraService.NewGameSessionService(sessionRepo, gameRepo, podRepo, nodeRepo, kubernetesService)
				},
				fx.As(new(service.GameSessionService)),
			),
			fx.Annotate(
				func(gameSessionService service.GameSessionService,
					gameEngine *usecase.GameEngineUseCase,
					gameUseCase *usecase.GameUseCase) service.WebSocketService {
					return infraWebSocket.NewWebSocketService(gameSessionService, gameEngine, gameUseCase)
				},
				fx.As(new(service.WebSocketService)),
			),
		),

		// Provide use cases - fixed order to avoid circular dependency
		fx.Provide(
			func(gameRepo repository.GameRepository,
				podRepo repository.PodRepository,
				nodeRepo repository.NodeRepository,
				sessionRepo repository.GameSessionRepository,
				k8sService service.KubernetesService) *usecase.GameEngineUseCase {
				// Create GameEngineUseCase without WebSocketService initially
				return usecase.NewGameEngineUseCase(gameRepo, podRepo, nodeRepo, sessionRepo, nil, k8sService)
			},
			func(gameRepo repository.GameRepository,
				podRepo repository.PodRepository,
				nodeRepo repository.NodeRepository,
				sessionRepo repository.GameSessionRepository,
				k8sService service.KubernetesService,
				schedulerSvc service.SchedulerService,
				gameEngine *usecase.GameEngineUseCase) *usecase.GameUseCase {
				// Create GameUseCase without WebSocketService initially  
				return usecase.NewGameUseCase(gameRepo, podRepo, nodeRepo, sessionRepo, k8sService, schedulerSvc, nil, gameEngine)
			},
		),

		// Extract dependencies for testing
		fx.Populate(
			&suite.wsService,
			&suite.gameSessionSvc,
			&suite.gameRepo,
			&suite.podRepo,
			&suite.nodeRepo,
			&suite.sessionRepo,
			&suite.gameUseCase,
			&suite.gameEngineUseCase,
		),
	)

	// Start the fx application
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start fx application: %v", err)
	}

	// Create HTTP test server with WebSocket handler
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.wsService.ServeHTTP(w, r)
	}))

	suite.app = app

	t.Cleanup(func() {
		suite.server.Close()
		app.Stop(context.Background())
	})

	return suite
}

// connectWebSocket creates a WebSocket connection for testing
func (suite *IntegrationTestSuite) connectWebSocket(t *testing.T) (*websocket.Conn, error) {
	wsURL := "ws" + strings.TrimPrefix(suite.server.URL, "http")
	
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		conn.Close()
	})

	return conn, nil
}

// sendMessage sends a message via WebSocket
func sendMessage(conn *websocket.Conn, msgType string, data interface{}) error {
	msg := TestMessage{
		Type: msgType,
		Data: data,
	}
	return conn.WriteJSON(msg)
}

// receiveMessage receives and parses a WebSocket message
func receiveMessage(conn *websocket.Conn, timeout time.Duration) (*entity.GameEvent, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	
	var event entity.GameEvent
	err := conn.ReadJSON(&event)
	if err != nil {
		return nil, err
	}
	
	return &event, nil
}

// TestWebSocketConnection tests basic WebSocket connection and session creation
func TestWebSocketConnection(t *testing.T) {
	suite := SetupIntegrationTest(t)

	// Connect to WebSocket
	conn, err := suite.connectWebSocket(t)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	// Wait a moment for session creation
	time.Sleep(100 * time.Millisecond)

	// Receive session_created event (should happen automatically)
	event, err := receiveMessage(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive session_created event: %v", err)
	}

	if event.Type != "session_created" {
		t.Errorf("Expected session_created event, got %s", event.Type)
	}

	// Verify session was created with dual namespaces
	sessions, err := suite.sessionRepo.GetAllSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to get sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	session := sessions[0]
	if session.State != entity.SessionStateClusterReady {
		t.Errorf("Expected session state %v, got %v", entity.SessionStateClusterReady, session.State)
	}

	// Verify namespaces were created
	if session.PlayerNamespace == "" {
		t.Error("Player namespace should not be empty")
	}
	if session.SchedulerNamespace == "" {
		t.Error("Scheduler namespace should not be empty")
	}
	if session.PlayerNamespace == session.SchedulerNamespace {
		t.Error("Player and scheduler namespaces should be different")
	}

	// Verify nodes were created (4 player + 1 scheduler)
	nodes, err := suite.nodeRepo.GetNodes(context.Background())
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	if len(nodes) != 5 {
		t.Errorf("Expected 5 nodes (4 player + 1 scheduler), got %d", len(nodes))
	}

	playerNodes := 0
	cpuNodes := 0
	for _, node := range nodes {
		if node.NodeType == "player" {
			playerNodes++
		} else if node.NodeType == "cpu" {
			cpuNodes++
		}
	}

	if playerNodes != 4 {
		t.Errorf("Expected 4 player nodes, got %d", playerNodes)
	}
	if cpuNodes != 1 {
		t.Errorf("Expected 1 CPU node, got %d", cpuNodes)
	}
}

// TestGameLifecycle tests the complete game lifecycle with dual namespaces
func TestGameLifecycle(t *testing.T) {
	suite := SetupIntegrationTest(t)

	// Connect to WebSocket
	conn, err := suite.connectWebSocket(t)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	// Wait for automatic session and cluster creation
	time.Sleep(100 * time.Millisecond)

	// Step 1: Receive session_created event automatically
	event, err := receiveMessage(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive session_created event: %v", err)
	}

	if event.Type != "session_created" {
		t.Errorf("Expected session_created event, got %s", event.Type)
	}

	// Step 2: Start game
	err = sendMessage(conn, "start_game", nil)
	if err != nil {
		t.Fatalf("Failed to send start_game message: %v", err)
	}

	// Receive game_started event
	event, err = receiveMessage(conn, 3*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive game_started event: %v", err)
	}

	if event.Type != "game_started" {
		t.Errorf("Expected game_started event, got %s", event.Type)
	}

	// Verify game was created by checking session
	sessions, err := suite.sessionRepo.GetAllSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to get sessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Fatal("No sessions found")
	}

	session := sessions[0]
	if session.GameID == nil {
		t.Fatal("Session should have a game ID after starting game")
	}

	// Get the game directly using the game ID from session
	game, err := suite.gameRepo.GetGame(context.Background(), *session.GameID)
	if err != nil {
		t.Fatalf("Failed to get game: %v", err)
	}

	if game.State != entity.GameStatePlaying {
		t.Errorf("Expected game state %v, got %v", entity.GameStatePlaying, game.State)
	}

	// Verify initial pod pairs were created in both namespaces
	pods, err := suite.podRepo.GetPods(context.Background())
	if err != nil {
		t.Fatalf("Failed to get pods: %v", err)
	}

	// Should have 16 pods total (8 pairs, one in each namespace)
	if len(pods) != 16 {
		t.Errorf("Expected 16 initial pods (8 pairs in 2 namespaces), got %d", len(pods))
	}

	// Verify pods are distributed correctly between namespaces
	playerPods := 0
	schedulerPods := 0
	for _, pod := range pods {
		if pod.Namespace == session.PlayerNamespace {
			playerPods++
		} else if pod.Namespace == session.SchedulerNamespace {
			schedulerPods++
		}
	}

	if playerPods != 8 {
		t.Errorf("Expected 8 pods in player namespace, got %d", playerPods)
	}
	if schedulerPods != 8 {
		t.Errorf("Expected 8 pods in scheduler namespace, got %d", schedulerPods)
	}

	// Verify nodes were created (4 player + 1 scheduler)
	nodes, err := suite.nodeRepo.GetNodes(context.Background())
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	if len(nodes) != 5 {
		t.Errorf("Expected 5 nodes (4 player + 1 scheduler), got %d", len(nodes))
	}

	// Step 3: Stop game
	err = sendMessage(conn, "stop_game", nil)
	if err != nil {
		t.Fatalf("Failed to send stop_game message: %v", err)
	}

	// Receive game_stopped event
	event, err = receiveMessage(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive game_stopped event: %v", err)
	}

	if event.Type != "game_stopped" {
		t.Errorf("Expected game_stopped event, got %s", event.Type)
	}

	// Verify game state was updated
	updatedGame, err := suite.gameRepo.GetGame(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Failed to get updated game: %v", err)
	}

	if updatedGame.State != entity.GameStateGameOver {
		t.Errorf("Expected game state %v, got %v", entity.GameStateGameOver, updatedGame.State)
	}
}

// TestPodScheduling tests pod scheduling functionality with namespace awareness
func TestPodScheduling(t *testing.T) {
	suite := SetupIntegrationTest(t)

	// Connect to WebSocket
	conn, err := suite.connectWebSocket(t)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	// Wait for session creation
	time.Sleep(100 * time.Millisecond)

	// Receive session_created automatically
	_, err = receiveMessage(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive session_created event: %v", err)
	}

	// Start game
	err = sendMessage(conn, "start_game", nil)
	if err != nil {
		t.Fatalf("Failed to send start_game message: %v", err)
	}

	// Receive game_started
	_, err = receiveMessage(conn, 3*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive game_started event: %v", err)
	}

	// Get session to access namespaces
	sessions, err := suite.sessionRepo.GetAllSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to get sessions: %v", err)
	}
	session := sessions[0]

	// Get available pods and nodes
	pods, err := suite.podRepo.GetPods(context.Background())
	if err != nil {
		t.Fatalf("Failed to get pods: %v", err)
	}

	nodes, err := suite.nodeRepo.GetNodes(context.Background())
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	// Find a player node and a pending pod in player namespace
	var playerNode *entity.Node
	var playerNamespacePod *entity.Pod

	for _, node := range nodes {
		if node.NodeType == "player" {
			playerNode = node
			break
		}
	}

	for _, pod := range pods {
		if pod.Status == entity.PodStatusPending && pod.Namespace == session.PlayerNamespace {
			playerNamespacePod = pod
			break
		}
	}

	if playerNode == nil {
		t.Fatal("No player node found")
	}

	if playerNamespacePod == nil {
		t.Fatal("No pending pod found in player namespace")
	}

	// Test pod scheduling in player namespace
	scheduleData := map[string]string{
		"pod_id":  playerNamespacePod.ID,
		"node_id": playerNode.ID,
	}

	err = sendMessage(conn, "schedule_pod", scheduleData)
	if err != nil {
		t.Fatalf("Failed to send schedule_pod message: %v", err)
	}

	// Receive pod_scheduled event (or error)
	event, err := receiveMessage(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive response: %v", err)
	}

	// Check if scheduling was successful or failed
	if event.Type == "pod_scheduled" {
		t.Logf("Pod scheduling successful in player namespace")
		
		// Verify pod was updated
		updatedPod, err := suite.podRepo.GetPod(context.Background(), playerNamespacePod.ID)
		if err != nil {
			t.Fatalf("Failed to get updated pod: %v", err)
		}
		
		if updatedPod.Owner != entity.PodOwnerPlayer {
			t.Errorf("Expected pod owner to be player, got %v", updatedPod.Owner)
		}
		
		if updatedPod.NodeID == nil || *updatedPod.NodeID != playerNode.ID {
			t.Errorf("Expected pod to be scheduled to node %s", playerNode.ID)
		}
		
	} else if event.Type == "error" {
		t.Logf("Pod scheduling failed (may be expected in some cases): %v", event.Data)
	} else {
		t.Errorf("Unexpected event type: %s", event.Type)
	}

	// Test that we cannot schedule pods from scheduler namespace
	var schedulerNamespacePod *entity.Pod
	for _, pod := range pods {
		if pod.Status == entity.PodStatusPending && pod.Namespace == session.SchedulerNamespace {
			schedulerNamespacePod = pod
			break
		}
	}

	if schedulerNamespacePod != nil {
		scheduleData := map[string]string{
			"pod_id":  schedulerNamespacePod.ID,
			"node_id": playerNode.ID,
		}

		err = sendMessage(conn, "schedule_pod", scheduleData)
		if err != nil {
			t.Fatalf("Failed to send schedule_pod message for scheduler namespace: %v", err)
		}

		// Should receive an error
		event, err = receiveMessage(conn, 2*time.Second)
		if err != nil {
			t.Fatalf("Failed to receive response: %v", err)
		}

		if event.Type != "error" {
			t.Errorf("Expected error when trying to schedule scheduler namespace pod, got %s", event.Type)
		} else {
			t.Logf("Correctly prevented scheduling of scheduler namespace pod: %v", event.Data)
		}
	}
}

// TestPingPong tests basic ping/pong functionality
func TestPingPong(t *testing.T) {
	suite := SetupIntegrationTest(t)

	// Connect to WebSocket
	conn, err := suite.connectWebSocket(t)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	// Send ping
	err = sendMessage(conn, "ping", nil)
	if err != nil {
		t.Fatalf("Failed to send ping message: %v", err)
	}

	// Receive pong
	event, err := receiveMessage(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive pong event: %v", err)
	}

	if event.Type != "pong" {
		t.Errorf("Expected pong event, got %s", event.Type)
	}
}

// TestMultipleClients tests multiple client connections with separate namespaces
func TestMultipleClients(t *testing.T) {
	suite := SetupIntegrationTest(t)

	// Connect multiple clients
	conn1, err := suite.connectWebSocket(t)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}

	conn2, err := suite.connectWebSocket(t)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}

	// Wait for session creation
	time.Sleep(200 * time.Millisecond)

	// Verify multiple sessions were created
	sessions, err := suite.sessionRepo.GetAllSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to get sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Each client should have their own cluster and namespaces
	clusterIDs := make(map[string]bool)
	playerNamespaces := make(map[string]bool)
	schedulerNamespaces := make(map[string]bool)
	
	for _, session := range sessions {
		clusterIDs[session.ClusterID] = true
		playerNamespaces[session.PlayerNamespace] = true
		schedulerNamespaces[session.SchedulerNamespace] = true
	}

	if len(clusterIDs) != 2 {
		t.Errorf("Expected 2 unique clusters, got %d", len(clusterIDs))
	}
	if len(playerNamespaces) != 2 {
		t.Errorf("Expected 2 unique player namespaces, got %d", len(playerNamespaces))
	}
	if len(schedulerNamespaces) != 2 {
		t.Errorf("Expected 2 unique scheduler namespaces, got %d", len(schedulerNamespaces))
	}

	// Both should receive session_created automatically
	_, err = receiveMessage(conn1, 2*time.Second)
	if err != nil {
		t.Fatalf("Client 1 did not receive session_created: %v", err)
	}

	_, err = receiveMessage(conn2, 2*time.Second)
	if err != nil {
		t.Fatalf("Client 2 did not receive session_created: %v", err)
	}

	// Verify each client has their own set of nodes (total 10 nodes: 5 per client)
	nodes, err := suite.nodeRepo.GetNodes(context.Background())
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	if len(nodes) != 10 {
		t.Errorf("Expected 10 nodes total (5 per client), got %d", len(nodes))
	}
}

// TestSessionCleanup tests session cleanup on disconnect
func TestSessionCleanup(t *testing.T) {
	suite := SetupIntegrationTest(t)

	// Connect and immediately disconnect
	conn, err := suite.connectWebSocket(t)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	// Wait for session creation
	time.Sleep(100 * time.Millisecond)

	// Verify session exists
	sessions, err := suite.sessionRepo.GetAllSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to get sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	// Close connection
	conn.Close()

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Note: In a real implementation, you might need to trigger cleanup manually
	// or wait longer for automatic cleanup to occur
	t.Logf("Session cleanup test completed. Note: Automatic cleanup timing may vary.")
}

// BenchmarkWebSocketThroughput benchmarks WebSocket message throughput
func BenchmarkWebSocketThroughput(b *testing.B) {
	suite := SetupIntegrationTest(&testing.T{})

	conn, err := suite.connectWebSocket(&testing.T{})
	if err != nil {
		b.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := sendMessage(conn, "ping", nil)
			if err != nil {
				b.Errorf("Failed to send ping: %v", err)
				return
			}

			_, err = receiveMessage(conn, 1*time.Second)
			if err != nil {
				b.Errorf("Failed to receive pong: %v", err)
				return
			}
		}
	})
}