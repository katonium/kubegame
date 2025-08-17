// Package usecase contains business logic and use cases for the game.
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

// Game configuration constants
const (
	GameDurationSeconds   = 300   // 5 minutes
	PointsPerPodPerSecond = 1     // Points earned per running pod per second
	GameTickInterval      = 1000  // Game tick interval in milliseconds
	EventTickInterval     = 15000 // Random event interval in milliseconds
	SchedulerTickInterval = 3000  // Kubernetes scheduler tick interval in milliseconds
)

// GameEngineUseCase handles game engine business logic.
type GameEngineUseCase struct {
	gameRepo     repository.GameRepository
	podRepo      repository.PodRepository
	nodeRepo     repository.NodeRepository
	wsService    service.WebSocketService
	k8sService   service.KubernetesService
	gameTickStop map[string]chan struct{} // Tracks active game tickers by game ID
}

// NewGameEngineUseCase creates a new game engine use case instance.
func NewGameEngineUseCase(
	gameRepo repository.GameRepository,
	podRepo repository.PodRepository,
	nodeRepo repository.NodeRepository,
	wsService service.WebSocketService,
	k8sService service.KubernetesService,
) *GameEngineUseCase {
	return &GameEngineUseCase{
		gameRepo:     gameRepo,
		podRepo:      podRepo,
		nodeRepo:     nodeRepo,
		wsService:    wsService,
		k8sService:   k8sService,
		gameTickStop: make(map[string]chan struct{}),
	}
}

// StartGame initializes and starts the game engine for a game session.
func (g *GameEngineUseCase) StartGame(ctx context.Context, gameID string) error {
	logger.Info(ctx, "Starting game engine for game %s", gameID)

	// Verify game exists
	game, err := g.gameRepo.GetGame(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game %s: %w", gameID, err)
	}

	if game.State != entity.GameStatePlaying {
		return fmt.Errorf("game %s is not in playing state", gameID)
	}

	// Create stop channel for this game
	stopCh := make(chan struct{})
	g.gameTickStop[gameID] = stopCh

	// Start game tickers
	go g.runGameTicker(ctx, gameID, stopCh)
	go g.runEventTicker(ctx, gameID, stopCh)
	go g.runSchedulerTicker(ctx, gameID, stopCh)

	logger.Info(ctx, "Game engine started for game %s", gameID)
	return nil
}

// StopGame stops the game engine and finalizes the game session.
func (g *GameEngineUseCase) StopGame(ctx context.Context, gameID string) error {
	logger.Info(ctx, "Stopping game engine for game %s", gameID)

	// Stop game tickers
	if stopCh, exists := g.gameTickStop[gameID]; exists {
		close(stopCh)
		delete(g.gameTickStop, gameID)
	}

	// Update game state
	game, err := g.gameRepo.GetGame(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game %s: %w", gameID, err)
	}

	game.State = entity.GameStateGameOver
	game.TimeLeft = 0
	game.UpdatedAt = time.Now()

	if err := g.gameRepo.UpdateGame(ctx, game); err != nil {
		return fmt.Errorf("failed to update game state: %w", err)
	}

	// Broadcast game over event
	event := &entity.GameEvent{
		Type:      "game_over",
		Data:      game,
		Timestamp: time.Now(),
	}
	g.wsService.BroadcastEvent(ctx, event)

	logger.Info(ctx, "Game %s ended - Player: %d, CPU: %d", gameID, game.PlayerScore, game.CPUScore)
	return nil
}

// HandlePodTermination handles random pod termination events.
func (g *GameEngineUseCase) HandlePodTermination(ctx context.Context, gameID string) error {
	logger.Debug(ctx, "Handling pod termination for game %s", gameID)

	// Get all running pods
	pods, err := g.podRepo.GetPodsByStatus(ctx, entity.PodStatusRunning)
	if err != nil {
		return fmt.Errorf("failed to get running pods: %w", err)
	}

	if len(pods) == 0 {
		logger.Debug(ctx, "No running pods to terminate")
		return nil
	}

	// Randomly select a pod to terminate
	podToTerminate := pods[rand.Intn(len(pods))]

	// Update pod status
	podToTerminate.Status = entity.PodStatusTerminated
	podToTerminate.NodeID = nil
	podToTerminate.Owner = entity.PodOwnerNone

	if err := g.podRepo.UpdatePod(ctx, podToTerminate); err != nil {
		return fmt.Errorf("failed to update terminated pod: %w", err)
	}

	// Delete pod from Kubernetes
	if err := g.k8sService.DeletePod(ctx, podToTerminate.ID); err != nil {
		logger.Warn(ctx, "Failed to delete pod %s from Kubernetes: %v", podToTerminate.ID, err)
	}

	// Broadcast pod termination
	g.wsService.BroadcastPodUpdate(ctx, podToTerminate)

	logger.Info(ctx, "Pod %s terminated", podToTerminate.ID)

	// Reset pod to pending after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		podToTerminate.Status = entity.PodStatusPending
		g.podRepo.UpdatePod(context.Background(), podToTerminate)
		g.wsService.BroadcastPodUpdate(context.Background(), podToTerminate)
	}()

	return nil
}

// GenerateRandomPod creates a new random pod for the game.
func (g *GameEngineUseCase) GenerateRandomPod(ctx context.Context, gameID string) (*entity.Pod, error) {
	logger.Debug(ctx, "Generating random pod for game %s", gameID)

	labels := []entity.PodLabel{
		entity.PodLabelBanana,
		entity.PodLabelChocolate,
		entity.PodLabelStrawberry,
		entity.PodLabelVanilla,
	}

	// Generate random pod
	pod := &entity.Pod{
		ID:    fmt.Sprintf("pod-%d", time.Now().UnixNano()),
		Name:  fmt.Sprintf("app-%d", rand.Intn(9999)),
		Label: labels[rand.Intn(len(labels))],
		Requirements: entity.ResourceRequirements{
			CPU:    rand.Intn(2) + 1, // 1-2 cores
			Memory: rand.Intn(3) + 1, // 1-3 GB
		},
		Status:    entity.PodStatusPending,
		Owner:     entity.PodOwnerNone,
		CreatedAt: time.Now(),
	}

	// Create pod in repository
	if err := g.podRepo.CreatePod(ctx, pod); err != nil {
		return nil, fmt.Errorf("failed to create random pod: %w", err)
	}

	// Broadcast new pod
	g.wsService.BroadcastPodUpdate(ctx, pod)

	logger.Info(ctx, "Generated random pod %s (%s) with %d CPU, %d GB memory",
		pod.ID, pod.Name, pod.Requirements.CPU, pod.Requirements.Memory)

	return pod, nil
}

// UpdateScores calculates and updates player and CPU scores.
func (g *GameEngineUseCase) UpdateScores(ctx context.Context, gameID string) error {
	logger.Debug(ctx, "Updating scores for game %s", gameID)

	// Get current game
	game, err := g.gameRepo.GetGame(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game: %w", err)
	}

	// Get running pods by owner
	playerPods, err := g.podRepo.GetPodsByOwner(ctx, entity.PodOwnerPlayer)
	if err != nil {
		return fmt.Errorf("failed to get player pods: %w", err)
	}

	cpuPods, err := g.podRepo.GetPodsByOwner(ctx, entity.PodOwnerCPU)
	if err != nil {
		return fmt.Errorf("failed to get CPU pods: %w", err)
	}

	// Count running pods
	runningPlayerPods := 0
	for _, pod := range playerPods {
		if pod.Status == entity.PodStatusRunning {
			runningPlayerPods++
		}
	}

	runningCPUPods := 0
	for _, pod := range cpuPods {
		if pod.Status == entity.PodStatusRunning {
			runningCPUPods++
		}
	}

	// Update scores
	game.PlayerScore += runningPlayerPods * PointsPerPodPerSecond
	game.CPUScore += runningCPUPods * PointsPerPodPerSecond
	game.UpdatedAt = time.Now()

	if err := g.gameRepo.UpdateGame(ctx, game); err != nil {
		return fmt.Errorf("failed to update game scores: %w", err)
	}

	logger.Debug(ctx, "Updated scores - Player: %d (+%d), CPU: %d (+%d)",
		game.PlayerScore, runningPlayerPods*PointsPerPodPerSecond,
		game.CPUScore, runningCPUPods*PointsPerPodPerSecond)

	return nil
}

// runGameTicker handles the main game loop ticker.
func (g *GameEngineUseCase) runGameTicker(ctx context.Context, gameID string, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(GameTickInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Update scores
			if err := g.UpdateScores(ctx, gameID); err != nil {
				logger.Error(ctx, "Failed to update scores for game %s: %v", gameID, err)
				continue
			}

			// Update time left
			game, err := g.gameRepo.GetGame(ctx, gameID)
			if err != nil {
				logger.Error(ctx, "Failed to get game %s: %v", gameID, err)
				continue
			}

			game.TimeLeft--
			if game.TimeLeft <= 0 {
				g.StopGame(ctx, gameID)
				return
			}

			game.UpdatedAt = time.Now()
			g.gameRepo.UpdateGame(ctx, game)

			// Broadcast game state update
			g.wsService.BroadcastGameState(ctx, game)

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
			// 50% chance to terminate a pod, 50% chance to generate a new pod
			if rand.Float32() < 0.5 {
				if err := g.HandlePodTermination(ctx, gameID); err != nil {
					logger.Error(ctx, "Failed to handle pod termination: %v", err)
				}
			} else {
				if _, err := g.GenerateRandomPod(ctx, gameID); err != nil {
					logger.Error(ctx, "Failed to generate random pod: %v", err)
				}
			}

		case <-stopCh:
			logger.Debug(ctx, "Event ticker stopped for game %s", gameID)
			return
		}
	}
}

// runSchedulerTicker handles automatic pod scheduling for CPU-owned pods.
func (g *GameEngineUseCase) runSchedulerTicker(ctx context.Context, gameID string, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(SchedulerTickInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get pending pods that are not owned by player
			allPods, err := g.podRepo.GetPodsByStatus(ctx, entity.PodStatusPending)
			if err != nil {
				logger.Error(ctx, "Failed to get pending pods: %v", err)
				continue
			}

			// Filter out player-owned pods
			availablePods := make([]*entity.Pod, 0)
			for _, pod := range allPods {
				if pod.Owner == entity.PodOwnerNone {
					availablePods = append(availablePods, pod)
				}
			}

			if len(availablePods) == 0 {
				continue
			}

			// Let Kubernetes scheduler handle these pods
			for _, pod := range availablePods {
				// Mark pod as CPU-owned and create in Kubernetes
				pod.Owner = entity.PodOwnerCPU
				pod.Status = entity.PodStatusScheduling

				if err := g.k8sService.CreatePod(ctx, pod); err != nil {
					logger.Warn(ctx, "Failed to create pod %s in Kubernetes: %v", pod.ID, err)
					pod.Status = entity.PodStatusFailed
				}

				if err := g.podRepo.UpdatePod(ctx, pod); err != nil {
					logger.Error(ctx, "Failed to update pod %s: %v", pod.ID, err)
				}

				g.wsService.BroadcastPodUpdate(ctx, pod)
			}

		case <-stopCh:
			logger.Debug(ctx, "Scheduler ticker stopped for game %s", gameID)
			return
		}
	}
}
