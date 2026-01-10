// Package usecase contains business logic and use cases for the game.
package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/katonium/kubegame/backend/adapter/kubernetes"
	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/message"
	"github.com/katonium/kubegame/backend/domain/repository"
	"github.com/katonium/kubegame/backend/generated"
	"github.com/katonium/kubegame/backend/util/logger"
)

// Game configuration constants
const (
	GameDurationSeconds   = 60    // 1 minute
	PointsPerPodPerSecond = 1     // Points earned per running pod per second
	GameTickInterval      = 1000  // Game tick interval in milliseconds
	EventTickInterval     = 8000  // Random event interval in milliseconds (reduced for more action)
	SchedulerTickInterval = 3000  // Kubernetes scheduler tick interval in milliseconds
	NodeEventInterval     = 25000 // Node create/delete event interval in milliseconds
	PodBurstInterval      = 12000 // Pod burst event interval in milliseconds
	NamespacePlayer       = "player"
	NamespaceScheduler    = "scheduler"
)

// GameEngineUseCaseIF defines the interface for game engine use case.
// FIXME: rename it into GameEngineInteractor
type GameEngineUseCaseIF interface {
	// InitializeGame sets up the game environment for a new game session.
	InitializeGame(ctx context.Context, gameID string, notifier Notifier) error

	// StartGame starts the game engine for a specific session.
	StartGame(ctx context.Context, gameID string) error

	// StopGame stops the game engine and notifies game result to player.
	// TODO: consider if we need this public method. This is seemed to be called internally only.
	// StopGame(ctx context.Context, gameID string) error

	// CleanupGame removes all resources associated with a game session.
	CleanupGame(ctx context.Context, gameID string) error

	// SchedulePod manually assigns a specific pod to a specific node in player namespace.
	SchedulePod(ctx context.Context, gameID, podID, nodeID string) error
}

// FIXME: remove this and apply dependency injection
var _ GameEngineUseCaseIF = &GameEngineUseCase{}

// GameEngineUseCase handles game engine business logic.
type GameEngineUseCase struct {
	k8sManager   kubernetes.ClusterManager
	gameTickStop map[string]chan struct{} // Tracks active game tickers by game ID
	gameRepo     repository.GameRepository
	notifiers    map[string]Notifier
}

// Notifier defines an interface to notify game events to client.
// TODO: move this
type Notifier interface {
	NotifyGameUpdate(ctx context.Context, event generated.GameUpdateEvent) error
	NotifyPodCreated(ctx context.Context, event generated.PodCreatedEvent) error
	NotifyPodScheduled(ctx context.Context, event generated.PodScheduledEvent) error
	NotifyGameOver(ctx context.Context, event generated.GameOverEvent) error
}

// NewGameEngineUseCase creates a new game engine use case instance.
func NewGameEngineUseCase(
	k8sManager kubernetes.ClusterManager,
	gameRepo repository.GameRepository,
) *GameEngineUseCase {
	return &GameEngineUseCase{
		k8sManager:   k8sManager,
		gameRepo:     gameRepo,
		gameTickStop: make(map[string]chan struct{}),
		notifiers:    make(map[string]Notifier),
	}
}

// InitializeGame prepares the game environment for a new game session.
func (uc *GameEngineUseCase) InitializeGame(ctx context.Context, gameID string, notifier Notifier) error {
	// create k8s cluster for this game session
	cluster, err := uc.k8sManager.CreateCluster(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to create k8s cluster for game %s: %w", gameID, err)
	}

	// create user and cpu namespaces
	if err := cluster.CreateNamespace(ctx, NamespacePlayer); err != nil {
		return fmt.Errorf("failed to create player namespace: %w", err)
	}
	if err := cluster.CreateNamespace(ctx, NamespaceScheduler); err != nil {
		return fmt.Errorf("failed to create scheduler namespace: %w", err)
	}

	// create game entry in repository
	game := &entity.Game{
		ID:        gameID,
		State:     entity.GameStatePreGame,
		TimeLeft:  GameDurationSeconds,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := uc.gameRepo.Put(ctx, game); err != nil {
		return fmt.Errorf("failed to create game entry: %w", err)
	}

	// register notifier
	uc.notifiers[gameID] = notifier

	return nil
}

func (g *GameEngineUseCase) SchedulePod(ctx context.Context, gameID, podID, nodeID string) error {
	logger.Info(ctx, "Scheduling pod %s to node %s in game %s", podID, nodeID, gameID)
	cluster, err := g.k8sManager.GetCluster(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get k8s cluster for game %s: %w", gameID, err)
	}

	// validate pod and node exist in player namespace
	pods, err := cluster.GetPods(ctx, NamespacePlayer)
	if err != nil {
		return fmt.Errorf("failed to list pods in player namespace: %w", err)
	}
	var podExists, nodeExists bool
	for _, pod := range pods {
		if pod.ID == podID {
			podExists = true
			break
		}
	}
	if !podExists {
		return fmt.Errorf("pod %s not found in player namespace", podID)
	}
	nodes, err := cluster.GetNodes(ctx, NamespacePlayer)
	if err != nil {
		return fmt.Errorf("failed to list nodes in player namespace: %w", err)
	}
	for _, node := range nodes {
		if node.ID == nodeID {
			nodeExists = true
			break
		}
	}
	if !nodeExists {
		return fmt.Errorf("node %s not found in player namespace", nodeID)
	}

	if err := cluster.SchedulePod(ctx, NamespacePlayer, podID, nodeID); err != nil {
		return fmt.Errorf("failed to schedule pod %s to node %s: %w", podID, nodeID, err)
	}
	return nil
}

// StartGame initializes and starts the game engine for a game session.
// If the game is already running, jsut return game state.
// If the gwsServiceame is over, return error.
func (g *GameEngineUseCase) StartGame(ctx context.Context, gameID string) error {
	logger.Info(ctx, "Starting game engine for game %s", gameID)

	// Verify game exists and is already running
	game, err := g.gameRepo.Get(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game %s: %w", gameID, err)
	}
	if game.State == entity.GameStatePlaying {
		return nil // already running
	}
	if game.State == entity.GameStateGameOver {
		return fmt.Errorf("game %s is already over", gameID)
	}

	// Create stop channel for this game
	stopCh := make(chan struct{})
	g.gameTickStop[gameID] = stopCh

	// turn game state into running
	game.State = entity.GameStatePlaying
	game.UpdatedAt = time.Now()
	if err := g.gameRepo.Update(ctx, game); err != nil {
		return fmt.Errorf("failed to update game state: %w", err)
	}

	// Generate initial pods and nodes
	if err := g.generateInitialPodsAndNodes(gameID); err != nil {
		return fmt.Errorf("failed to generate initial pods: %w", err)
	}

	// Start game tickers
	go g.runGameTicker(ctx, gameID, stopCh)
	go g.runEventTicker(ctx, gameID, stopCh)
	go g.runSchedulerTicker(ctx, gameID, stopCh)

	logger.Info(ctx, "Game engine started for game %s", gameID)
	return nil
}

func (g *GameEngineUseCase) generateInitialPodsAndNodes(gameID string) error {
	cluster, err := g.k8sManager.GetCluster(context.Background(), gameID)
	if err != nil {
		return fmt.Errorf("failed to get k8s cluster for game %s: %w", gameID, err)
	}
	// Create initial nodes in player namespace
	// 2 nodes for player
	for i := 1; i <= 3; i++ {
		id := uuid.New().String()
		node := &entity.Node{
			ID:        id,
			Name:      fmt.Sprintf("player-node-%s", id[:8]),
			Capacity:  entity.NodeCapacity{CPU: 8, Memory: 16},
			Used:      entity.NodeCapacity{CPU: 0, Memory: 0},
			Namespace: NamespacePlayer,
		}
		if err := cluster.CreateNode(context.Background(), NamespacePlayer, node); err != nil {
			return fmt.Errorf("failed to create player node: %w", err)
		}
	}
	// 2 nodes for cpu
	for i := 1; i <= 3; i++ {
		id := uuid.New().String()
		node := &entity.Node{
			ID:        id,
			Name:      fmt.Sprintf("cpu-node-%s", id[:8]),
			Capacity:  entity.NodeCapacity{CPU: 8, Memory: 16},
			Used:      entity.NodeCapacity{CPU: 0, Memory: 0},
			Namespace: NamespaceScheduler,
		}
		if err := cluster.CreateNode(context.Background(), NamespacePlayer, node); err != nil {
			return fmt.Errorf("failed to create player node: %w", err)
		}
	}

	// Create initial pods in both namespaces
	// 3 pods for player
	for i := 1; i <= 3; i++ {
		id := uuid.New().String()
		pod := &entity.Pod{
			ID:    id,
			Name:  fmt.Sprintf("player-pod-%s", id[:8]),
			Label: entity.PodLabelVanilla, // TODO
			Requirements: entity.ResourceRequirements{
				CPU:    rand.Intn(2) + 1, // 1-2 cores
				Memory: rand.Intn(3) + 1, // 1-3 GB
			},
			Status:    entity.PodStatusPending,
			NodeID:    nil,
			Namespace: NamespacePlayer,
			CreatedAt: time.Now(),
		}
		if err := cluster.CreatePod(context.Background(), NamespacePlayer, pod); err != nil {
			return fmt.Errorf("failed to create player pod: %w", err)
		}
	}
	// 3 pods for cpu
	for i := 1; i <= 3; i++ {
		id := uuid.New().String()
		pod := &entity.Pod{
			ID:    id,
			Name:  fmt.Sprintf("cpu-pod-%s", id[:8]),
			Label: entity.PodLabelChocolate, // TODO
			Requirements: entity.ResourceRequirements{
				CPU:    rand.Intn(2) + 1, // 1-2 cores
				Memory: rand.Intn(3) + 1, // 1-3 GB
			},
			Status:    entity.PodStatusPending,
			NodeID:    nil,
			Namespace: NamespaceScheduler,
			CreatedAt: time.Now(),
		}
		if err := cluster.CreatePod(context.Background(), NamespaceScheduler, pod); err != nil {
			return fmt.Errorf("failed to create cpu pod: %w", err)
		}
	}
	logger.Debug(context.Background(), "Initial pods and nodes created for game %s", gameID)
	return nil
}

// StopGame stops the game engine and finalizes the game session.
// This is assumed to be called by ticker when game is over or player disconnects.
func (g *GameEngineUseCase) stopGame(ctx context.Context, gameID string) error {
	logger.Info(ctx, "Stopping game engine for game %s", gameID)

	// Stop game tickers
	if stopCh, exists := g.gameTickStop[gameID]; exists {
		close(stopCh)
		delete(g.gameTickStop, gameID)
	}

	// Update game state
	game, err := g.gameRepo.Get(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game %s: %w", gameID, err)
	}

	game.State = entity.GameStateGameOver
	game.TimeLeft = 0
	game.UpdatedAt = time.Now()

	if err := g.gameRepo.Put(ctx, game); err != nil {
		return fmt.Errorf("failed to update game state: %w", err)
	}

	// notify game over event
	playerCluster, schedulerCluster, err := g.getClusterInfo(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get cluster info for game %s: %w", gameID, err)
	}

	event := &generated.GameOverEvent{
		Type: message.EventTypeGameOver,
		Data: generated.GameInfo{
			PlayerScore:      game.PlayerScore,
			CpuScore:         game.CPUScore,
			PlayerCluster:    playerCluster,
			SchedulerCluster: schedulerCluster,
			TimeLeft:         game.TimeLeft,
		},
		Timestamp: time.Now(),
	}
	notifier, exists := g.notifiers[gameID]
	if !exists {
		return fmt.Errorf("no notifier found for game %s", gameID)
	}
	if err := notifier.NotifyGameOver(ctx, *event); err != nil {
		return fmt.Errorf("failed to notify game over event: %w", err)
	}

	logger.Info(ctx, "Game %s ended - Player: %d, CPU: %d", gameID, game.PlayerScore, game.CPUScore)
	return nil
}

func (g *GameEngineUseCase) getClusterInfo(ctx context.Context, gameID string) (playerCluster, schedulerCluster generated.ClusterInfo, err error) {
	cluster, err := g.k8sManager.GetCluster(ctx, gameID)
	if err != nil {
		return playerCluster, schedulerCluster, fmt.Errorf("failed to get k8s cluster for game %s: %w", gameID, err)
	}
	playerPods, err := cluster.GetPods(ctx, NamespacePlayer)
	if err != nil {
		return playerCluster, schedulerCluster, fmt.Errorf("failed to list player pods: %w", err)
	}
	schedulerPods, err := cluster.GetPods(ctx, NamespaceScheduler)
	if err != nil {
		return playerCluster, schedulerCluster, fmt.Errorf("failed to list scheduler pods: %w", err)
	}
	allNodes, err := cluster.GetNodes(ctx, kubernetes.AllNamespaces)
	if err != nil {
		return playerCluster, schedulerCluster, fmt.Errorf("failed to get nodes: %w", err)
	}
	var playerNodes, schedulerNodes []*entity.Node
	for _, node := range allNodes {
		if node.Namespace == NamespacePlayer {
			playerNodes = append(playerNodes, node)
		} else if node.Namespace == NamespaceScheduler {
			schedulerNodes = append(schedulerNodes, node)
		} else {
			logger.Warn(ctx, "Node %s has unknown namespace label: %s", node.ID, node.Namespace)
		}
	}
	playerCluster = generated.ClusterInfo{
		Pods:  make([]generated.PodInfo, len(playerPods)),
		Nodes: make([]generated.NodeInfo, len(playerNodes)),
	}
	schedulerCluster = generated.ClusterInfo{
		Pods:  make([]generated.PodInfo, len(schedulerPods)),
		Nodes: make([]generated.NodeInfo, len(schedulerNodes)),
	}
	for i, pod := range playerPods {
		playerCluster.Pods[i] = g.getPodInfo(pod)
	}
	for i, pod := range schedulerPods {
		schedulerCluster.Pods[i] = g.getPodInfo(pod)
	}
	for i, node := range playerNodes {
		playerCluster.Nodes[i] = g.getNodeInfo(node, true)
	}
	for i, node := range schedulerNodes {
		schedulerCluster.Nodes[i] = g.getNodeInfo(node, false)
	}
	return playerCluster, schedulerCluster, nil
}

func (g *GameEngineUseCase) getPodInfo(pod *entity.Pod) generated.PodInfo {
	return generated.PodInfo{
		Id:       pod.ID,
		Name:     pod.Name,
		Label:    generated.PodLabels(pod.Label),
		Affinity: generated.Affinity{
			// TODO
		},
		NodeID: pod.NodeID,
		Requirements: generated.ResourceRequirements{
			Cpu:    pod.Requirements.CPU,
			Memory: pod.Requirements.Memory,
		},
		Status: generated.PodInfoStatus(pod.Status),
	}
}

const (
	NodeInfoTypePlayer = "player"
	NodeInfoTypeCPU    = "cpu"
)

func (g *GameEngineUseCase) getNodeInfo(node *entity.Node, isPlayer bool) generated.NodeInfo {
	nodeType := generated.Scheduler
	if isPlayer {
		nodeType = generated.Player
	}

	return generated.NodeInfo{
		Id:   node.ID,
		Name: node.Name,
		Type: nodeType,
		Capacity: generated.ResourceRequirements{
			Cpu:    node.Capacity.CPU,
			Memory: node.Capacity.Memory,
		},
		Used: generated.ResourceRequirements{
			Cpu:    node.Used.CPU,
			Memory: node.Used.Memory,
		},
	}
}

func (g *GameEngineUseCase) CleanupGame(ctx context.Context, gameID string) error {
	return nil
}

// GenerateRandomPod creates a new random pod pair for both namespaces in the game session.
func (g *GameEngineUseCase) generateRandomPod(ctx context.Context, gameID string) error {
	logger.Debug(ctx, "Generating random pod pair for game %s", gameID)

	panic("not implemented yet")

	return nil
}

// UpdateScores calculates and updates player and CPU scores for a specific session.
func (g *GameEngineUseCase) UpdateScores(ctx context.Context, gameID string) error {
	logger.Debug(ctx, "Updating scores for game %s", gameID)

	// Get current game
	// game, err := g.gameRepo.Get(ctx, gameID)
	// if err != nil {
	// 	return fmt.Errorf("failed to get game: %w", err)
	// }

	// Get all pods for this session
	// cluster, err := g.k8sManager.GetCluster(ctx, gameID)
	// if err != nil {
	// 	return fmt.Errorf("failed to get k8s cluster for game %s: %w", gameID, err)
	// }
	// allPods, err := cluster.ListPods(ctx, "")
	// if err != nil {
	// 	return fmt.Errorf("failed to get pods: %w", err)
	// }

	// // Count running pods by namespace and owner
	// runningPlayerPods := 0
	// runningCPUPods := 0

	// for _, pod := range allPods {
	// 	if pod.Status == entity.PodStatusRunning {
	// 		// Player namespace pods owned by player
	// 		if pod.Namespace == session.PlayerNamespace && pod.Owner == entity.PodOwnerPlayer {
	// 			runningPlayerPods++
	// 		}
	// 		// Scheduler namespace pods owned by CPU
	// 		if pod.Namespace == session.SchedulerNamespace && pod.Owner == entity.PodOwnerCPU {
	// 			runningCPUPods++
	// 		}
	// 	}
	// }

	// // Update scores
	// game.PlayerScore += runningPlayerPods * PointsPerPodPerSecond
	// game.CPUScore += runningCPUPods * PointsPerPodPerSecond
	// game.UpdatedAt = time.Now()

	// if err := g.gameRepo.Update(ctx, game); err != nil {
	// 	return fmt.Errorf("failed to update game scores: %w", err)
	// }

	// logger.Debug(ctx, "Updated scores - Player: %d (+%d), CPU: %d (+%d)",
	// 	game.PlayerScore, runningPlayerPods*PointsPerPodPerSecond,
	// 	game.CPUScore, runningCPUPods*PointsPerPodPerSecond)

	return nil
}

// runGameTicker handles the main game loop ticker.
func (g *GameEngineUseCase) runGameTicker(ctx context.Context, gameID string, stopCh chan struct{}) {
	interval := time.Duration(GameTickInterval * time.Millisecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Update scores
			logger.Debug(ctx, "Game tick for game %s", gameID)
			if err := g.UpdateScores(ctx, gameID); err != nil {
				logger.Error(ctx, "Failed to update scores for game %s: %v", gameID, err)
				continue
			}

			// Update time left
			game, err := g.gameRepo.Get(ctx, gameID)
			if err != nil {
				logger.Error(ctx, "Failed to get game %s: %v", gameID, err)
				continue
			}
			game.TimeLeft-- // FIXME: This should be ticker interval or actual elapsed time
			if game.TimeLeft <= 0 {
				g.stopGame(ctx, gameID)
				return
			}

			game.UpdatedAt = time.Now()
			g.gameRepo.Update(ctx, game)

			// notify game state update to specific session
			event := generated.GameUpdateEvent{
				Type: message.EventTypeGameUpdate,
				Data: generated.GameInfo{
					PlayerScore:      game.PlayerScore,
					CpuScore:         game.CPUScore,
					PlayerCluster:    generated.ClusterInfo{},
					SchedulerCluster: generated.ClusterInfo{},
					TimeLeft:         game.TimeLeft,
				},
				Timestamp: time.Now(),
			}
			notifier, exists := g.notifiers[gameID]
			if !exists {
				logger.Error(ctx, "No notifier found for game %s", gameID)
				continue
			}
			if err := notifier.NotifyGameUpdate(ctx, event); err != nil {
				logger.Error(ctx, "Failed to notify game update for game %s: %v", gameID, err)
			}
		case <-stopCh:
			logger.Debug(ctx, "Game ticker stopped for game %s", gameID)
			return
		}
	}
}

// runEventTicker handles random game events.
func (g *GameEngineUseCase) runEventTicker(ctx context.Context, gameID string, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(EventTickInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 50% chance to generate a new pod
			if rand.Float32() < 0.5 {
				if err := g.generateRandomPod(ctx, gameID); err != nil {
					logger.Error(ctx, "Failed to generate random pod: %v", err)
				}
			}
		case <-stopCh:
			logger.Debug(ctx, "Event ticker stopped for game %s", gameID)
			return
		}
	}
}

// runSchedulerTicker handles automatic pod scheduling for CPU-owned pods in scheduler namespace.
func (g *GameEngineUseCase) runSchedulerTicker(ctx context.Context, gameID string, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(SchedulerTickInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// TODO

			// // Get all pods for this session
			// allPods, err := g.podRepo.GetPods(ctx)
			// if err != nil {
			// 	logger.Error(ctx, "Failed to get pods: %v", err)
			// 	continue
			// }

			// // Find pending pods in scheduler namespace that are unowned
			// var availablePods []*entity.Pod
			// for _, pod := range allPods {
			// 	if pod.Namespace == session.SchedulerNamespace &&
			// 		pod.Status == entity.PodStatusPending &&
			// 		pod.Owner == entity.PodOwnerNone {
			// 		availablePods = append(availablePods, pod)
			// 	}
			// }

			// if len(availablePods) == 0 {
			// 	continue
			// }

			// // Let Kubernetes scheduler handle these pods
			// for _, pod := range availablePods {
			// 	// Mark pod as CPU-owned and create in Kubernetes
			// 	pod.Owner = entity.PodOwnerCPU
			// 	pod.Status = entity.PodStatusScheduling

			// 	if err := g.k8sManager.CreatePod(ctx, pod); err != nil {
			// 		logger.Warn(ctx, "Failed to create pod %s in Kubernetes: %v", pod.ID, err)
			// 		pod.Status = entity.PodStatusFailed
			// 	}

			// 	if err := g.podRepo.UpdatePod(ctx, pod); err != nil {
			// 		logger.Error(ctx, "Failed to update pod %s: %v", pod.ID, err)
			// 	}

			// 	// Send update to specific session
			// 	if g.wsService != nil {
			// 		g.wsService.SendEventToClient(ctx, session.ConnectionID, &entity.GameEvent{
			// 			Type:      message.EventTypePodCreated,
			// 			Data:      pod,
			// 			Timestamp: time.Now(),
			// 		})
			// 	}

			// 	logger.Debug(ctx, "CPU scheduler claimed pod %s in namespace %s", pod.ID, pod.Namespace)
			// }

		case <-stopCh:
			logger.Debug(ctx, "Scheduler ticker stopped for game %s", gameID)
			return
		}
	}
}
