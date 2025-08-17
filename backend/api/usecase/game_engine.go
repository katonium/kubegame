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
	GameDurationSeconds   = 60    // 1 minute
	PointsPerPodPerSecond = 1     // Points earned per running pod per second
	GameTickInterval      = 1000  // Game tick interval in milliseconds
	EventTickInterval     = 8000  // Random event interval in milliseconds (reduced for more action)
	SchedulerTickInterval = 3000  // Kubernetes scheduler tick interval in milliseconds
	NodeEventInterval     = 25000 // Node create/delete event interval in milliseconds
	PodBurstInterval      = 12000 // Pod burst event interval in milliseconds
)

// GameEngineUseCase handles game engine business logic.
type GameEngineUseCase struct {
	gameRepo     repository.GameRepository
	podRepo      repository.PodRepository
	nodeRepo     repository.NodeRepository
	sessionRepo  repository.GameSessionRepository
	wsService    service.WebSocketService
	k8sService   service.KubernetesService
	gameTickStop map[string]chan struct{} // Tracks active game tickers by game ID
}

// NewGameEngineUseCase creates a new game engine use case instance.
func NewGameEngineUseCase(
	gameRepo repository.GameRepository,
	podRepo repository.PodRepository,
	nodeRepo repository.NodeRepository,
	sessionRepo repository.GameSessionRepository,
	wsService service.WebSocketService,
	k8sService service.KubernetesService,
) *GameEngineUseCase {
	return &GameEngineUseCase{
		gameRepo:     gameRepo,
		podRepo:      podRepo,
		nodeRepo:     nodeRepo,
		sessionRepo:  sessionRepo,
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
	if g.wsService != nil {
		g.wsService.BroadcastEvent(ctx, event)
	}

	logger.Info(ctx, "Game %s ended - Player: %d, CPU: %d", gameID, game.PlayerScore, game.CPUScore)
	return nil
}

// HandlePodTermination handles random pod termination events for a specific session.
func (g *GameEngineUseCase) HandlePodTermination(ctx context.Context, gameID string) error {
	logger.Debug(ctx, "Handling pod termination for game %s", gameID)

	// Find session for this game
	session, err := g.getSessionByGameID(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to find session for game %s: %w", gameID, err)
	}

	// Get all running pods for this session from both namespaces
	allPods, err := g.podRepo.GetPods(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	var runningPods []*entity.Pod
	for _, pod := range allPods {
		if pod.Status == entity.PodStatusRunning && 
		   (pod.Namespace == session.PlayerNamespace || pod.Namespace == session.SchedulerNamespace) {
			runningPods = append(runningPods, pod)
		}
	}

	if len(runningPods) == 0 {
		logger.Debug(ctx, "No running pods to terminate in session %s", session.SessionID)
		return nil
	}

	// Randomly select a pod to terminate
	podToTerminate := runningPods[rand.Intn(len(runningPods))]

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

	// Send pod update to specific session
	if g.wsService != nil {
		g.wsService.SendEventToClient(ctx, session.ConnectionID, &entity.GameEvent{
			Type:      "pod_update",
			Data:      podToTerminate,
			Timestamp: time.Now(),
		})
	}

	logger.Info(ctx, "Pod %s terminated in namespace %s for session %s", 
		podToTerminate.ID, podToTerminate.Namespace, session.SessionID)

	// Reset pod to pending after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		podToTerminate.Status = entity.PodStatusPending
		g.podRepo.UpdatePod(context.Background(), podToTerminate)
		if g.wsService != nil {
			g.wsService.SendEventToClient(context.Background(), session.ConnectionID, &entity.GameEvent{
				Type:      "pod_update",
				Data:      podToTerminate,
				Timestamp: time.Now(),
			})
		}
	}()

	return nil
}

// GenerateRandomPod creates a new random pod pair for both namespaces in the game session.
func (g *GameEngineUseCase) GenerateRandomPod(ctx context.Context, gameID string) (*entity.Pod, error) {
	logger.Debug(ctx, "Generating random pod pair for game %s", gameID)

	// Find session for this game
	session, err := g.getSessionByGameID(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to find session for game %s: %w", gameID, err)
	}

	labels := []entity.PodLabel{
		entity.PodLabelBanana,
		entity.PodLabelChocolate,
		entity.PodLabelStrawberry,
		entity.PodLabelVanilla,
	}

	// Define pod specification once
	podSpec := struct {
		name         string
		label        entity.PodLabel
		requirements entity.ResourceRequirements
	}{
		name:  fmt.Sprintf("app-%d", rand.Intn(9999)),
		label: labels[rand.Intn(len(labels))],
		requirements: entity.ResourceRequirements{
			CPU:    rand.Intn(2) + 1, // 1-2 cores
			Memory: rand.Intn(3) + 1, // 1-3 GB
		},
	}

	timestamp := time.Now().UnixNano()

	// Create pod in player namespace
	playerPod := &entity.Pod{
		ID:           fmt.Sprintf("%s-player-pod-%d", gameID, timestamp),
		Name:         podSpec.name,
		Label:        podSpec.label,
		Requirements: podSpec.requirements,
		Status:       entity.PodStatusPending,
		Owner:        entity.PodOwnerNone,
		Namespace:    session.PlayerNamespace,
		CreatedAt:    time.Now(),
	}

	if err := g.podRepo.CreatePod(ctx, playerPod); err != nil {
		return nil, fmt.Errorf("failed to create player pod: %w", err)
	}

	// Create identical pod in scheduler namespace
	schedulerPod := &entity.Pod{
		ID:           fmt.Sprintf("%s-scheduler-pod-%d", gameID, timestamp),
		Name:         podSpec.name,
		Label:        podSpec.label,
		Requirements: podSpec.requirements,
		Status:       entity.PodStatusPending,
		Owner:        entity.PodOwnerNone,
		Namespace:    session.SchedulerNamespace,
		CreatedAt:    time.Now(),
	}

	if err := g.podRepo.CreatePod(ctx, schedulerPod); err != nil {
		return nil, fmt.Errorf("failed to create scheduler pod: %w", err)
	}

	// Send both pod updates to the specific session
	if g.wsService != nil {
		g.wsService.SendEventToClient(ctx, session.ConnectionID, &entity.GameEvent{
			Type:      "pod_update",
			Data:      playerPod,
			Timestamp: time.Now(),
		})

		g.wsService.SendEventToClient(ctx, session.ConnectionID, &entity.GameEvent{
			Type:      "pod_update",
			Data:      schedulerPod,
			Timestamp: time.Now(),
		})
	}

	logger.Info(ctx, "Generated random pod pair %s (%s) with %d CPU, %d GB memory for session %s in both namespaces",
		podSpec.name, podSpec.label, podSpec.requirements.CPU, podSpec.requirements.Memory, session.SessionID)

	return playerPod, nil
}

// UpdateScores calculates and updates player and CPU scores for a specific session.
func (g *GameEngineUseCase) UpdateScores(ctx context.Context, gameID string) error {
	logger.Debug(ctx, "Updating scores for game %s", gameID)

	// Get current game
	game, err := g.gameRepo.GetGame(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game: %w", err)
	}

	// Find the session for this game
	session, err := g.getSessionByGameID(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to find session for game %s: %w", gameID, err)
	}

	// Get all pods for this session
	allPods, err := g.podRepo.GetPods(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	// Count running pods by namespace and owner
	runningPlayerPods := 0
	runningCPUPods := 0

	for _, pod := range allPods {
		if pod.Status == entity.PodStatusRunning {
			// Player namespace pods owned by player
			if pod.Namespace == session.PlayerNamespace && pod.Owner == entity.PodOwnerPlayer {
				runningPlayerPods++
			}
			// Scheduler namespace pods owned by CPU
			if pod.Namespace == session.SchedulerNamespace && pod.Owner == entity.PodOwnerCPU {
				runningCPUPods++
			}
		}
	}

	// Update scores
	game.PlayerScore += runningPlayerPods * PointsPerPodPerSecond
	game.CPUScore += runningCPUPods * PointsPerPodPerSecond
	game.UpdatedAt = time.Now()

	if err := g.gameRepo.UpdateGame(ctx, game); err != nil {
		return fmt.Errorf("failed to update game scores: %w", err)
	}

	logger.Debug(ctx, "Updated scores - Player: %d (+%d from %s), CPU: %d (+%d from %s)",
		game.PlayerScore, runningPlayerPods*PointsPerPodPerSecond, session.PlayerNamespace,
		game.CPUScore, runningCPUPods*PointsPerPodPerSecond, session.SchedulerNamespace)

	return nil
}

// runGameTicker handles the main game loop ticker.
func (g *GameEngineUseCase) runGameTicker(ctx context.Context, gameID string, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(GameTickInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if session still exists
			_, err := g.getSessionByGameID(ctx, gameID)
			if err != nil {
				logger.Debug(ctx, "Session for game %s no longer exists, stopping game ticker", gameID)
				return
			}

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

			// Broadcast game state update to specific session
			session, err := g.getSessionByGameID(ctx, gameID)
			if err != nil {
				logger.Error(ctx, "Failed to get session for game %s: %v", gameID, err)
			} else if g.wsService != nil {
				g.wsService.SendEventToClient(ctx, session.ConnectionID, &entity.GameEvent{
					Type:      "game_state_update",
					Data:      game,
					Timestamp: game.UpdatedAt,
				})
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

// runSchedulerTicker handles automatic pod scheduling for CPU-owned pods in scheduler namespace.
func (g *GameEngineUseCase) runSchedulerTicker(ctx context.Context, gameID string, stopCh chan struct{}) {
	ticker := time.NewTicker(time.Duration(SchedulerTickInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Find session for this game
			session, err := g.getSessionByGameID(ctx, gameID)
			if err != nil {
				logger.Error(ctx, "Failed to find session for game %s: %v", gameID, err)
				continue
			}

			// Get all pods for this session
			allPods, err := g.podRepo.GetPods(ctx)
			if err != nil {
				logger.Error(ctx, "Failed to get pods: %v", err)
				continue
			}

			// Find pending pods in scheduler namespace that are unowned
			var availablePods []*entity.Pod
			for _, pod := range allPods {
				if pod.Namespace == session.SchedulerNamespace && 
				   pod.Status == entity.PodStatusPending && 
				   pod.Owner == entity.PodOwnerNone {
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

				// Send update to specific session
				if g.wsService != nil {
					g.wsService.SendEventToClient(ctx, session.ConnectionID, &entity.GameEvent{
						Type:      "pod_update",
						Data:      pod,
						Timestamp: time.Now(),
					})
				}

				logger.Debug(ctx, "CPU scheduler claimed pod %s in namespace %s", pod.ID, pod.Namespace)
			}

		case <-stopCh:
			logger.Debug(ctx, "Scheduler ticker stopped for game %s", gameID)
			return
		}
	}
}

// Helper methods for session-scoped operations

// getSessionByGameID finds the session that owns a specific game
func (g *GameEngineUseCase) getSessionByGameID(ctx context.Context, gameID string) (*entity.GameSession, error) {
	sessions, err := g.sessionRepo.GetAllSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	for _, session := range sessions {
		if session.GameID != nil && *session.GameID == gameID {
			return session, nil
		}
	}

	return nil, fmt.Errorf("no session found for game %s", gameID)
}

// getSessionPodsByOwner gets pods owned by a specific owner within a session
func (g *GameEngineUseCase) getSessionPodsByOwner(ctx context.Context, session *entity.GameSession, owner entity.PodOwner) ([]*entity.Pod, error) {
	allPods, err := g.podRepo.GetPodsByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}

	var sessionPods []*entity.Pod
	for _, pod := range allPods {
		if g.isPodInSession(pod, session) {
			sessionPods = append(sessionPods, pod)
		}
	}

	return sessionPods, nil
}

// isPodInSession checks if a pod belongs to a specific session
func (g *GameEngineUseCase) isPodInSession(pod *entity.Pod, session *entity.GameSession) bool {
	if session.GameID == nil {
		return false
	}
	// Check if pod ID contains the game ID (which contains session ID)
	return len(pod.ID) >= len(*session.GameID) && 
		   pod.ID[:len(*session.GameID)] == *session.GameID
}

// getSessionPods gets pods in a specific session with optional status filter
func (g *GameEngineUseCase) getSessionPods(ctx context.Context, session *entity.GameSession, status entity.PodStatus) ([]*entity.Pod, error) {
	var allPods []*entity.Pod
	var err error
	
	if status != "" {
		allPods, err = g.podRepo.GetPodsByStatus(ctx, status)
	} else {
		allPods, err = g.podRepo.GetPods(ctx)
	}
	
	if err != nil {
		return nil, err
	}

	var sessionPods []*entity.Pod
	for _, pod := range allPods {
		if g.isPodInSession(pod, session) {
			sessionPods = append(sessionPods, pod)
		}
	}

	return sessionPods, nil
}
