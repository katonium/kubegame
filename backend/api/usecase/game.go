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
	gameRepo     repository.GameRepository
	podRepo      repository.PodRepository
	nodeRepo     repository.NodeRepository
	k8sService   service.KubernetesService
	schedulerSvc service.SchedulerService
	wsService    service.WebSocketService
	gameEngine   *GameEngineUseCase
}

func NewGameUseCase(
	gameRepo repository.GameRepository,
	podRepo repository.PodRepository,
	nodeRepo repository.NodeRepository,
	k8sService service.KubernetesService,
	schedulerSvc service.SchedulerService,
	wsService service.WebSocketService,
	gameEngine *GameEngineUseCase,
) *GameUseCase {
	return &GameUseCase{
		gameRepo:     gameRepo,
		podRepo:      podRepo,
		nodeRepo:     nodeRepo,
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

	// Initialize game nodes and pods in Kubernetes cluster
	if err := uc.initializeGameState(ctx, gameID); err != nil {
		return fmt.Errorf("failed to initialize game state: %w", err)
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
	uc.wsService.BroadcastEvent(ctx, event)

	return nil
}

func (uc *GameUseCase) SchedulePodToNode(ctx context.Context, podID, nodeID, owner string) error {
	logger.Info(ctx, "Player requesting to schedule pod %s to node %s", podID, nodeID)

	// Get pod from repository
	pod, err := uc.podRepo.GetPod(ctx, podID)
	if err != nil {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Get node from repository
	node, err := uc.nodeRepo.GetNode(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Only allow player to schedule to player nodes
	if owner == "player" && node.NodeType != "player" {
		return fmt.Errorf("players can only schedule to player nodes")
	}

	// Update pod to set node selector for player scheduling
	pod.Status = entity.PodStatusScheduling
	if owner == "player" {
		pod.Owner = entity.PodOwnerPlayer
		// Create pod in Kubernetes with nodeSelector for specific node
		if err := uc.k8sService.CreatePod(ctx, pod); err != nil {
			return fmt.Errorf("failed to create pod in Kubernetes: %w", err)
		}
		// Force schedule to the selected node
		if err := uc.k8sService.SchedulePod(ctx, podID, nodeID); err != nil {
			pod.Status = entity.PodStatusFailed
			logger.Warn(ctx, "Failed to schedule pod %s to node %s: %v", podID, nodeID, err)
		}
	}

	if err := uc.podRepo.UpdatePod(ctx, pod); err != nil {
		return fmt.Errorf("failed to update pod: %w", err)
	}

	// Broadcast pod update
	uc.wsService.BroadcastPodUpdate(ctx, pod)

	return nil
}

func (uc *GameUseCase) SchedulePendingPodsWithScheduler(ctx context.Context) error {
	// Get all pending pods that don't have an owner (available for Kubernetes scheduler)
	pods, err := uc.podRepo.GetPodsByStatus(ctx, entity.PodStatusPending)
	if err != nil {
		return fmt.Errorf("failed to get pending pods: %w", err)
	}

	for _, pod := range pods {
		// Only schedule pods that are not owned by player
		if pod.Owner == entity.PodOwnerNone {
			logger.Debug(ctx, "Letting Kubernetes scheduler handle pod: %s", pod.ID)

			// Create pod in Kubernetes without nodeSelector - let scheduler decide
			pod.Owner = entity.PodOwnerCPU
			pod.Status = entity.PodStatusScheduling

			if err := uc.k8sService.CreatePod(ctx, pod); err != nil {
				logger.Error(ctx, "Failed to create pod %s in Kubernetes: %v", pod.ID, err)
				continue
			}

			if err := uc.podRepo.UpdatePod(ctx, pod); err != nil {
				logger.Error(ctx, "Failed to update pod %s: %v", pod.ID, err)
				continue
			}

			// Broadcast pod update
			uc.wsService.BroadcastPodUpdate(ctx, pod)
		}
	}

	return nil
}

func (uc *GameUseCase) GetGameState(ctx context.Context, gameID string) (*entity.Game, []*entity.Pod, []*entity.Node, error) {
	game, err := uc.gameRepo.GetGame(ctx, gameID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get game: %w", err)
	}

	pods, err := uc.podRepo.GetPods(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get pods: %w", err)
	}

	nodes, err := uc.nodeRepo.GetNodes(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	return game, pods, nodes, nil
}

func (uc *GameUseCase) initializeGameState(ctx context.Context, gameID string) error {
	// Create player nodes in Kubernetes cluster
	playerNodes := []*entity.Node{
		{ID: "player-node-1", Name: "worker-01", Capacity: entity.NodeCapacity{CPU: 4, Memory: 8}, NodeType: "player"},
		{ID: "player-node-2", Name: "worker-02", Capacity: entity.NodeCapacity{CPU: 6, Memory: 12}, NodeType: "player"},
		{ID: "player-node-3", Name: "worker-03", Capacity: entity.NodeCapacity{CPU: 2, Memory: 4}, NodeType: "player"},
	}

	// Create CPU nodes (managed by Kubernetes scheduler)
	cpuNodes := []*entity.Node{
		{ID: "cpu-node-1", Name: "scheduler-01", Capacity: entity.NodeCapacity{CPU: 8, Memory: 16}, NodeType: "cpu"},
		{ID: "cpu-node-2", Name: "scheduler-02", Capacity: entity.NodeCapacity{CPU: 4, Memory: 8}, NodeType: "cpu"},
		{ID: "cpu-node-3", Name: "scheduler-03", Capacity: entity.NodeCapacity{CPU: 6, Memory: 12}, NodeType: "cpu"},
	}

	// Create nodes in Kubernetes cluster and repository
	allNodes := append(playerNodes, cpuNodes...)
	for _, node := range allNodes {
		if err := uc.k8sService.CreateNode(ctx, node); err != nil {
			return fmt.Errorf("failed to create node %s in Kubernetes: %w", node.ID, err)
		}
		if err := uc.nodeRepo.CreateNode(ctx, node); err != nil {
			return fmt.Errorf("failed to create node %s in repository: %w", node.ID, err)
		}
	}

	// Create initial pods in repository (but not yet in Kubernetes)
	initialPods := uc.generateInitialPods()
	for _, pod := range initialPods {
		if err := uc.podRepo.CreatePod(ctx, pod); err != nil {
			return fmt.Errorf("failed to create pod %s: %w", pod.ID, err)
		}
	}

	return nil
}

func (uc *GameUseCase) generateInitialPods() []*entity.Pod {
	labels := []entity.PodLabel{
		entity.PodLabelBanana,
		entity.PodLabelChocolate,
		entity.PodLabelStrawberry,
		entity.PodLabelVanilla,
	}

	pods := make([]*entity.Pod, 0, 8)
	for i := 0; i < 8; i++ {
		pod := &entity.Pod{
			ID:    fmt.Sprintf("initial-pod-%d", i+1),
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
