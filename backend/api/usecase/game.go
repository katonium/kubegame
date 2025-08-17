package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/repository"
	"github.com/katonium/kubegame/backend/domain/service"
	"github.com/katonium/kubegame/backend/util/logger"
)

type GameUseCase struct {
	gameRepo       repository.GameRepository
	podRepo        repository.PodRepository
	nodeRepo       repository.NodeRepository
	sessionRepo    repository.GameSessionRepository
	k8sService     service.KubernetesService
	schedulerSvc   service.SchedulerService
	wsService      service.WebSocketService
	gameEngine     *GameEngineUseCase
}

func NewGameUseCase(
	gameRepo repository.GameRepository,
	podRepo repository.PodRepository,
	nodeRepo repository.NodeRepository,
	sessionRepo repository.GameSessionRepository,
	k8sService service.KubernetesService,
	schedulerSvc service.SchedulerService,
	wsService service.WebSocketService,
	gameEngine *GameEngineUseCase,
) *GameUseCase {
	return &GameUseCase{
		gameRepo:     gameRepo,
		podRepo:      podRepo,
		nodeRepo:     nodeRepo,
		sessionRepo:  sessionRepo,
		k8sService:   k8sService,
		schedulerSvc: schedulerSvc,
		wsService:    wsService,
		gameEngine:   gameEngine,
	}
}

func (uc *GameUseCase) StartGame(ctx context.Context, gameID string) error {
	logger.Info(ctx, "Starting game: %s", gameID)

	// Create game entity
	game := &entity.Game{
		ID:          gameID,
		State:       entity.GameStatePlaying,
		PlayerScore: 0,
		CPUScore:    0,
		TimeLeft:    GameDurationSeconds,
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.gameRepo.CreateGame(ctx, game); err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	// Initialize game state with initial pods for the session
	if err := uc.initializeGamePods(ctx, gameID); err != nil {
		return fmt.Errorf("failed to initialize game pods: %w", err)
	}

	// Start game engine
	if err := uc.gameEngine.StartGame(ctx, gameID); err != nil {
		return fmt.Errorf("failed to start game engine: %w", err)
	}

	// Broadcast game started event
	event := &entity.GameEvent{
		Type:      "game_started",
		Data:      game,
		Timestamp: time.Now(),
	}
	if uc.wsService != nil {
		uc.wsService.BroadcastEvent(ctx, event)
	}

	return nil
}

func (uc *GameUseCase) StopGame(ctx context.Context, gameID string) error {
	logger.Info(ctx, "Stopping game: %s", gameID)

	// Get game
	game, err := uc.gameRepo.GetGame(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game: %w", err)
	}

	// Stop game engine
	if err := uc.gameEngine.StopGame(ctx, gameID); err != nil {
		logger.Error(ctx, "Failed to stop game engine: %v", err)
	}

	// Update game state
	game.State = entity.GameStateGameOver
	game.TimeLeft = 0
	game.UpdatedAt = time.Now()

	if err := uc.gameRepo.UpdateGame(ctx, game); err != nil {
		return fmt.Errorf("failed to update game: %w", err)
	}

	// Broadcast game stopped event
	event := &entity.GameEvent{
		Type:      "game_stopped",
		Data:      game,
		Timestamp: time.Now(),
	}
	if uc.wsService != nil {
		uc.wsService.BroadcastEvent(ctx, event)
	}

	logger.Info(ctx, "Game %s stopped successfully", gameID)
	return nil
}

func (uc *GameUseCase) SchedulePodToNode(ctx context.Context, podID, nodeID, connectionID string) error {
	logger.Info(ctx, "Connection %s requesting to schedule pod %s to node %s", connectionID, podID, nodeID)

	// Get session to verify ownership
	session, err := uc.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Get pod from repository
	pod, err := uc.podRepo.GetPod(ctx, podID)
	if err != nil {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Verify pod is in player namespace
	if pod.Namespace != session.PlayerNamespace {
		return fmt.Errorf("can only schedule pods in player namespace %s", session.PlayerNamespace)
	}

	// Get node from repository and verify it belongs to this session
	node, err := uc.nodeRepo.GetNode(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Verify node belongs to this session's cluster
	if !uc.isNodeInSession(node, session) {
		return fmt.Errorf("node %s does not belong to session %s", nodeID, session.SessionID)
	}

	// Only allow player to schedule to player nodes
	if node.NodeType != "player" {
		return fmt.Errorf("players can only schedule to player nodes")
	}

	// Check resource capacity
	if !uc.canNodeAccommodatePod(ctx, node, pod) {
		return fmt.Errorf("node %s does not have enough capacity for pod %s", nodeID, podID)
	}

	// Update pod to set node selector for player scheduling
	pod.Status = entity.PodStatusScheduling
	pod.Owner = entity.PodOwnerPlayer
	pod.NodeID = &nodeID

	// Create pod in Kubernetes with nodeSelector for specific node
	if err := uc.k8sService.CreatePod(ctx, pod); err != nil {
		return fmt.Errorf("failed to create pod in Kubernetes: %w", err)
	}

	// Force schedule to the selected node
	if err := uc.k8sService.SchedulePod(ctx, podID, nodeID); err != nil {
		pod.Status = entity.PodStatusFailed
		logger.Warn(ctx, "Failed to schedule pod %s to node %s: %v", podID, nodeID, err)
	} else {
		pod.Status = entity.PodStatusRunning
	}

	if err := uc.podRepo.UpdatePod(ctx, pod); err != nil {
		return fmt.Errorf("failed to update pod: %w", err)
	}

	// Broadcast pod update to the specific session
	if uc.wsService != nil {
		uc.wsService.SendEventToClient(ctx, connectionID, &entity.GameEvent{
			Type:      "pod_update",
			Data:      pod,
			Timestamp: time.Now(),
		})
	}

	logger.Info(ctx, "Player scheduled pod %s to node %s in namespace %s", 
		podID, nodeID, pod.Namespace)

	return nil
}

func (uc *GameUseCase) SchedulePendingPodsWithScheduler(ctx context.Context, connectionID string) error {
	// Get session to scope the operation
	session, err := uc.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Get all pending pods that don't have an owner (available for Kubernetes scheduler)
	pods, err := uc.podRepo.GetPodsByStatus(ctx, entity.PodStatusPending)
	if err != nil {
		return fmt.Errorf("failed to get pending pods: %w", err)
	}

	// Get CPU nodes for this session
	cpuNodes, err := uc.getSessionNodes(ctx, session, "cpu")
	if err != nil {
		return fmt.Errorf("failed to get CPU nodes for session: %w", err)
	}

	for _, pod := range pods {
		// Only schedule pods that are not owned by player and belong to this session's game
		if pod.Owner == entity.PodOwnerNone && uc.isPodInSession(pod, session) {
			logger.Debug(ctx, "Letting Kubernetes scheduler handle pod: %s", pod.ID)

			// Schedule to a random CPU node for this session
			if len(cpuNodes) > 0 {
				selectedNode := cpuNodes[rand.Intn(len(cpuNodes))]
				
				pod.Owner = entity.PodOwnerCPU
				pod.Status = entity.PodStatusScheduling
				pod.NodeID = &selectedNode.ID

				if err := uc.k8sService.CreatePod(ctx, pod); err != nil {
					logger.Error(ctx, "Failed to create pod %s in Kubernetes: %v", pod.ID, err)
					continue
				}

				// Simulate successful scheduling to CPU node
				pod.Status = entity.PodStatusRunning

				if err := uc.podRepo.UpdatePod(ctx, pod); err != nil {
					logger.Error(ctx, "Failed to update pod %s: %v", pod.ID, err)
					continue
				}

				// Broadcast pod update to the specific session
				if uc.wsService != nil {
					uc.wsService.SendEventToClient(ctx, connectionID, &entity.GameEvent{
						Type:      "pod_update",
						Data:      pod,
						Timestamp: time.Now(),
					})
				}
			}
		}
	}

	return nil
}

func (uc *GameUseCase) GetGameState(ctx context.Context, connectionID string) (*entity.Game, []*entity.Pod, []*entity.Node, error) {
	// Get session
	session, err := uc.sessionRepo.GetSession(ctx, connectionID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get session: %w", err)
	}

	var game *entity.Game
	if session.GameID != nil {
		game, err = uc.gameRepo.GetGame(ctx, *session.GameID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get game: %w", err)
		}
	}

	// Get session-specific resources
	nodes, err := uc.getSessionNodes(ctx, session, "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	pods, err := uc.getSessionPods(ctx, session)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get pods: %w", err)
	}

	return game, pods, nodes, nil
}

func (uc *GameUseCase) initializeGamePods(ctx context.Context, gameID string) error {
	// Note: Initial pods are created by GameSessionService.createInitialPods
	// This method is kept for interface compatibility
	logger.Debug(ctx, "Game pods initialized for %s", gameID)
	return nil
}

func (uc *GameUseCase) generateInitialPods(gameID string) []*entity.Pod {
	labels := []entity.PodLabel{
		entity.PodLabelBanana,
		entity.PodLabelChocolate,
		entity.PodLabelStrawberry,
		entity.PodLabelVanilla,
	}

	pods := make([]*entity.Pod, 0, 8)
	for i := 0; i < 8; i++ {
		pod := &entity.Pod{
			ID:    fmt.Sprintf("%s-pod-%d", gameID, i+1),
			Name:  fmt.Sprintf("app-%d", i+1),
			Label: labels[rand.Intn(len(labels))],
			Requirements: entity.ResourceRequirements{
				CPU:    rand.Intn(2) + 1, // 1-2 cores
				Memory: rand.Intn(3) + 1, // 1-3 GB
			},
			Status:    entity.PodStatusPending,
			Owner:     entity.PodOwnerNone, // Available for both player and Kubernetes scheduler
			CreatedAt: time.Now(),
		}
		pods = append(pods, pod)
	}

	return pods
}

// Helper functions for session-scoped operations

func (uc *GameUseCase) isNodeInSession(node *entity.Node, session *entity.GameSession) bool {
	// Check if node ID contains the session's cluster ID
	return len(node.ID) >= len(session.ClusterID) && 
		   node.ID[len(node.ID)-len(session.ClusterID):] == session.ClusterID
}

func (uc *GameUseCase) isPodInSession(pod *entity.Pod, session *entity.GameSession) bool {
	// Check if pod ID contains the game ID (which contains session ID)
	if session.GameID == nil {
		return false
	}
	return len(pod.ID) >= len(*session.GameID) && 
		   pod.ID[:len(*session.GameID)] == *session.GameID
}

func (uc *GameUseCase) getSessionNodes(ctx context.Context, session *entity.GameSession, nodeType string) ([]*entity.Node, error) {
	allNodes, err := uc.nodeRepo.GetNodes(ctx)
	if err != nil {
		return nil, err
	}

	var sessionNodes []*entity.Node
	for _, node := range allNodes {
		if uc.isNodeInSession(node, session) {
			if nodeType == "" || node.NodeType == nodeType {
				sessionNodes = append(sessionNodes, node)
			}
		}
	}

	return sessionNodes, nil
}

func (uc *GameUseCase) getSessionPods(ctx context.Context, session *entity.GameSession) ([]*entity.Pod, error) {
	allPods, err := uc.podRepo.GetPods(ctx)
	if err != nil {
		return nil, err
	}

	var sessionPods []*entity.Pod
	for _, pod := range allPods {
		// Check if pod belongs to this session's namespaces
		if pod.Namespace == session.PlayerNamespace || pod.Namespace == session.SchedulerNamespace {
			sessionPods = append(sessionPods, pod)
		}
	}

	return sessionPods, nil
}

// canNodeAccommodatePod checks if a node has enough capacity to accommodate a pod
func (uc *GameUseCase) canNodeAccommodatePod(ctx context.Context, node *entity.Node, newPod *entity.Pod) bool {
	// Get all pods currently running on this node
	allPods, err := uc.podRepo.GetPods(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to get pods for capacity check: %v", err)
		return false
	}

	// Calculate current resource usage on the node
	usedCPU := 0
	usedMemory := 0
	for _, pod := range allPods {
		if pod.NodeID != nil && *pod.NodeID == node.ID && pod.Status == entity.PodStatusRunning {
			usedCPU += pod.Requirements.CPU
			usedMemory += pod.Requirements.Memory
		}
	}

	// Check if there's enough capacity for the new pod
	availableCPU := node.Capacity.CPU - usedCPU
	availableMemory := node.Capacity.Memory - usedMemory

	hasCapacity := availableCPU >= newPod.Requirements.CPU && availableMemory >= newPod.Requirements.Memory

	logger.Debug(ctx, "Node %s capacity check: CPU %d/%d (need %d), Memory %d/%d (need %d) -> %t",
		node.ID, usedCPU, node.Capacity.CPU, newPod.Requirements.CPU,
		usedMemory, node.Capacity.Memory, newPod.Requirements.Memory, hasCapacity)

	return hasCapacity
}
