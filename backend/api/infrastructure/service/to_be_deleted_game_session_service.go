package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/repository"
	"github.com/katonium/kubegame/backend/domain/service"
	"github.com/katonium/kubegame/backend/util/logger"
)

type gameSessionService struct {
	sessionRepo       repository.GameSessionRepository
	gameRepo          repository.GameRepository
	podRepo           repository.PodRepository
	nodeRepo          repository.NodeRepository
	kubernetesService service.KubernetesService
	gameEngineUseCase GameEngineUseCase
}

type GameEngineUseCase interface {
	StartGame(ctx context.Context, gameID string) error
	StopGame(ctx context.Context, gameID string) error
}

func NewGameSessionService(
	sessionRepo repository.GameSessionRepository,
	gameRepo repository.GameRepository,
	podRepo repository.PodRepository,
	nodeRepo repository.NodeRepository,
	kubernetesService service.KubernetesService,
) service.GameSessionService {
	return &gameSessionService{
		sessionRepo:       sessionRepo,
		gameRepo:          gameRepo,
		podRepo:           podRepo,
		nodeRepo:          nodeRepo,
		kubernetesService: kubernetesService,
	}
}

func (s *gameSessionService) CreateSession(ctx context.Context, connectionID string) (*entity.GameSession, error) {
	logger.Info(ctx, "Creating new session for connection %s", connectionID)

	// Check if session already exists
	if existing, _ := s.sessionRepo.GetSession(ctx, connectionID); existing != nil {
		return existing, nil
	}

	// Create new session
	session := &entity.GameSession{
		ConnectionID:       connectionID,
		SessionID:          uuid.New().String(),
		State:              entity.SessionStateConnected,
		ClusterID:          fmt.Sprintf("cluster-%s", connectionID),
		PlayerNamespace:    fmt.Sprintf("player-%s", connectionID),
		SchedulerNamespace: fmt.Sprintf("scheduler-%s", connectionID),
		GameID:             nil,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		LastActivity:       time.Now(),
	}

	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	logger.Info(ctx, "Created session %s for connection %s with cluster %s (namespaces: %s, %s)",
		session.SessionID, connectionID, session.ClusterID, session.PlayerNamespace, session.SchedulerNamespace)
	return session, nil
}

func (s *gameSessionService) GetSession(ctx context.Context, connectionID string) (*entity.GameSession, error) {
	return s.sessionRepo.GetSession(ctx, connectionID)
}

func (s *gameSessionService) UpdateSessionActivity(ctx context.Context, connectionID string) error {
	session, err := s.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.LastActivity = time.Now()
	session.UpdatedAt = time.Now()

	return s.sessionRepo.UpdateSession(ctx, session)
}

func (s *gameSessionService) CloseSession(ctx context.Context, connectionID string) error {
	logger.Info(ctx, "Closing session for connection %s", connectionID)

	session, err := s.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Stop any active game
	if session.GameID != nil {
		if err := s.StopGame(ctx, connectionID); err != nil {
			logger.Error(ctx, "Failed to stop game for session %s: %v", session.SessionID, err)
		}
	}

	// Delete cluster
	if err := s.DeleteCluster(ctx, connectionID); err != nil {
		logger.Error(ctx, "Failed to delete cluster for session %s: %v", session.SessionID, err)
	}

	// Update session state
	session.State = entity.SessionStateDisconnected
	session.UpdatedAt = time.Now()
	if err := s.sessionRepo.UpdateSession(ctx, session); err != nil {
		logger.Error(ctx, "Failed to update session state: %v", err)
	}

	// Delete session
	return s.sessionRepo.DeleteSession(ctx, connectionID)
}

func (s *gameSessionService) CreateCluster(ctx context.Context, connectionID string) error {
	logger.Info(ctx, "Creating cluster for connection %s", connectionID)

	session, err := s.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Create fake Kubernetes client for this session
	// In a real implementation, you might create namespace isolation

	// Create initial nodes for the cluster - 4 player nodes + 1 k8s scheduler node
	playerNodes := []*entity.Node{
		{
			ID:   fmt.Sprintf("player-node-1-%s", session.ClusterID),
			Name: "worker-01",
			Capacity: entity.NodeCapacity{
				CPU:    4,
				Memory: 8,
			},
			// NodeType: "player",
		},
		{
			ID:   fmt.Sprintf("player-node-2-%s", session.ClusterID),
			Name: "worker-02",
			Capacity: entity.NodeCapacity{
				CPU:    6,
				Memory: 12,
			},
			// NodeType: "player",
		},
		{
			ID:   fmt.Sprintf("player-node-3-%s", session.ClusterID),
			Name: "worker-03",
			Capacity: entity.NodeCapacity{
				CPU:    2,
				Memory: 4,
			},
			// NodeType: "player",
		},
		{
			ID:   fmt.Sprintf("player-node-4-%s", session.ClusterID),
			Name: "worker-04",
			Capacity: entity.NodeCapacity{
				CPU:    8,
				Memory: 16,
			},
			// NodeType: "player",
		},
	}

	cpuNodes := []*entity.Node{
		{
			ID:   fmt.Sprintf("cpu-node-1-%s", session.ClusterID),
			Name: "scheduler-01",
			Capacity: entity.NodeCapacity{
				CPU:    12,
				Memory: 24,
			},
			// NodeType: "cpu",
		},
	}

	// Create nodes in repository
	allNodes := append(playerNodes, cpuNodes...)
	for _, node := range allNodes {
		if err := s.nodeRepo.CreateNode(ctx, node); err != nil {
			return fmt.Errorf("failed to create node %s: %w", node.ID, err)
		}
		logger.Info(ctx, "Created node %s (%s) with capacity %d CPU, %d GB memory", node.ID, node.Name, node.Capacity.CPU, node.Capacity.Memory)
	}

	// Update session state
	session.State = entity.SessionStateClusterReady
	session.UpdatedAt = time.Now()

	return s.sessionRepo.UpdateSession(ctx, session)
}

func (s *gameSessionService) DeleteCluster(ctx context.Context, connectionID string) error {
	logger.Info(ctx, "Deleting cluster for connection %s", connectionID)

	session, err := s.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Get all nodes for this cluster
	allNodes, err := s.nodeRepo.GetNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	// Delete nodes belonging to this cluster
	for _, node := range allNodes {
		if node.ID[len(node.ID)-len(session.ClusterID):] == session.ClusterID {
			if err := s.nodeRepo.DeleteNode(ctx, node.ID); err != nil {
				logger.Error(ctx, "Failed to delete node %s: %v", node.ID, err)
			}
		}
	}

	// Get all pods for this session and delete them
	allPods, err := s.podRepo.GetPods(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to get pods for cleanup: %v", err)
	} else {
		for _, pod := range allPods {
			// Delete pods belonging to this session's namespaces
			if pod.Namespace == session.PlayerNamespace || pod.Namespace == session.SchedulerNamespace {
				if err := s.podRepo.DeletePod(ctx, pod.ID); err != nil {
					logger.Error(ctx, "Failed to delete pod %s from namespace %s: %v", pod.ID, pod.Namespace, err)
				}
			}
		}
	}

	logger.Info(ctx, "Deleted cluster %s with namespaces %s and %s for connection %s",
		session.ClusterID, session.PlayerNamespace, session.SchedulerNamespace, connectionID)
	return nil
}

func (s *gameSessionService) StartGame(ctx context.Context, connectionID string) error {
	logger.Info(ctx, "Starting game for connection %s", connectionID)

	session, err := s.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.State != entity.SessionStateClusterReady {
		return fmt.Errorf("cluster not ready for session %s", session.SessionID)
	}

	// Create a unique game ID for this session
	gameID := fmt.Sprintf("game-%s", session.SessionID)

	// Create game entity
	game := &entity.Game{
		ID:          gameID,
		State:       entity.GameStatePlaying,
		PlayerScore: 0,
		CPUScore:    0,
		TimeLeft:    60, // 1 minutes
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.gameRepo.Put(ctx, game); err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	// Create initial pods for the game
	if err := s.createInitialPods(ctx, gameID); err != nil {
		logger.Error(ctx, "Failed to create initial pods: %v", err)
	}

	// Update session
	session.GameID = &gameID
	session.State = entity.SessionStatePlaying
	session.UpdatedAt = time.Now()

	if err := s.sessionRepo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Note: Game engine will be started separately by the use case layer
	// This avoids circular dependencies while still providing the core functionality

	logger.Info(ctx, "Started game %s for session %s", gameID, session.SessionID)
	return nil
}

func (s *gameSessionService) StopGame(ctx context.Context, connectionID string) error {
	logger.Info(ctx, "Stopping game for connection %s", connectionID)

	session, err := s.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.GameID == nil {
		return nil // No active game
	}

	gameID := *session.GameID

	// Stop game engine if available
	if s.gameEngineUseCase != nil {
		if err := s.gameEngineUseCase.StopGame(ctx, gameID); err != nil {
			logger.Error(ctx, "Failed to stop game engine: %v", err)
		}
	}

	// Update game state
	game, err := s.gameRepo.Get(ctx, gameID)
	if err == nil {
		game.State = entity.GameStateGameOver
		game.TimeLeft = 0
		game.UpdatedAt = time.Now()
		s.gameRepo.Update(ctx, game)
	}

	// Update session
	session.GameID = nil
	session.State = entity.SessionStateClusterReady
	session.UpdatedAt = time.Now()

	if err := s.sessionRepo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	logger.Info(ctx, "Stopped game %s for session %s", gameID, session.SessionID)
	return nil
}

func (s *gameSessionService) RestartGame(ctx context.Context, connectionID string) error {
	logger.Info(ctx, "Restarting game for connection %s", connectionID)

	// Stop current game if running
	if err := s.StopGame(ctx, connectionID); err != nil {
		logger.Error(ctx, "Failed to stop current game: %v", err)
	}

	// Start new game
	return s.StartGame(ctx, connectionID)
}

func (s *gameSessionService) GetSessionState(ctx context.Context, connectionID string) (*entity.GameSession, error) {
	return s.sessionRepo.GetSession(ctx, connectionID)
}

func (s *gameSessionService) CleanupInactiveSessions(ctx context.Context) error {
	logger.Debug(ctx, "Cleaning up inactive sessions")

	// Clean up sessions inactive for more than 30 minutes
	timeout := 30 * time.Minute
	return s.sessionRepo.CleanupInactiveSessions(ctx, timeout)
}

// createInitialPods creates the initial set of pods for a new game session
// Creates identical pods in both player and scheduler namespaces for competition
func (s *gameSessionService) createInitialPods(ctx context.Context, gameID string) error {
	logger.Debug(ctx, "Creating initial pods for game %s", gameID)

	// Get session from gameID to access namespaces
	sessionID := gameID[5:] // Remove "game-" prefix to get session ID
	var session *entity.GameSession
	allSessions, err := s.sessionRepo.GetAllSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}

	for _, s := range allSessions {
		if s.SessionID == sessionID {
			session = s
			break
		}
	}

	if session == nil {
		return fmt.Errorf("session not found for game %s", gameID)
	}

	labels := []entity.PodLabel{
		entity.PodLabelBanana,
		entity.PodLabelChocolate,
		entity.PodLabelStrawberry,
		entity.PodLabelVanilla,
	}

	// Create 8 initial pods in both namespaces (total 16 pods)
	for i := 0; i < 8; i++ {
		// Define pod specification once
		podSpec := struct {
			name         string
			label        entity.PodLabel
			requirements entity.ResourceRequirements
		}{
			name:  fmt.Sprintf("app-%d", i+1),
			label: labels[rand.Intn(len(labels))],
			requirements: entity.ResourceRequirements{
				CPU:    rand.Intn(2) + 1, // 1-2 cores
				Memory: rand.Intn(3) + 1, // 1-3 GB
			},
		}

		// Create pod in player namespace
		playerPod := &entity.Pod{
			ID:           fmt.Sprintf("%s-player-pod-%d", gameID, i+1),
			Name:         podSpec.name,
			Label:        podSpec.label,
			Requirements: podSpec.requirements,
			Status:       entity.PodStatusPending,
			Namespace:    session.PlayerNamespace,
			CreatedAt:    time.Now(),
		}

		if err := s.podRepo.CreatePod(ctx, playerPod); err != nil {
			return fmt.Errorf("failed to create player pod %s: %w", playerPod.ID, err)
		}

		// Create identical pod in scheduler namespace
		schedulerPod := &entity.Pod{
			ID:           fmt.Sprintf("%s-scheduler-pod-%d", gameID, i+1),
			Name:         podSpec.name,
			Label:        podSpec.label,
			Requirements: podSpec.requirements,
			Status:       entity.PodStatusPending,
			Namespace:    session.SchedulerNamespace,
			CreatedAt:    time.Now(),
		}

		if err := s.podRepo.CreatePod(ctx, schedulerPod); err != nil {
			return fmt.Errorf("failed to create scheduler pod %s: %w", schedulerPod.ID, err)
		}

		logger.Debug(ctx, "Created pod pair %s (%s) with %d CPU, %d GB memory in both namespaces",
			podSpec.name, podSpec.label, podSpec.requirements.CPU, podSpec.requirements.Memory)
	}

	logger.Info(ctx, "Created 8 pod pairs (16 total pods) for game %s in namespaces %s and %s",
		gameID, session.PlayerNamespace, session.SchedulerNamespace)
	return nil
}
