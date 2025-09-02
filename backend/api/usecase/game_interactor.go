package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/util/logger"
)

// GameSession stores information about a user's game session and associated resources.
type GameSession struct {
	// ID is the unique identifier for the game session, typically the WebSocket connection ID.
	ID string

	// GameID is the unique identifier for the game session
	// This is not the same as the connection ID.
	// Because a user can retry game so have multiple game sessions in one connection.
	GameID string

	// CpuScore Current CPU score
	CpuScore int

	// PlayerScore Current player score
	PlayerScore int

	// TimeLeft Time remaining in seconds
	TimeLeft time.Time
}

type GameInteractor struct {
	gameEngine *GameEngineUseCase
}

func NewGameInteractor(
	gameEngine *GameEngineUseCase,
) *GameInteractor {
	return &GameInteractor{
		gameEngine: gameEngine,
	}
}

// get game id
// FIXME: currently use connection ID as game ID
// After we implemented persistent game info, user retry game record confilcts
// To avoid this, we need to generate a new game ID for each game session
// But for now, we keep it simple and use connection ID as game ID
func (uc *GameInteractor) getGameID(connID string) string {
	return connID
}

// generate game id
// FIXME: currently use connection ID as game ID
// After we implemented persistent game info, user retry game record confilcts
// To avoid this, we need to generate a new game ID for each game session
// But for now, we keep it simple and use connection ID as game ID
func (uc *GameInteractor) generateGameID(connID string) string {
	return connID
}

// InitializeGame prepares the game environment for a new game session.
func (uc *GameInteractor) InitializeGame(ctx context.Context, connID string, notifier Notifier) error {
	gameID := uc.generateGameID(connID)
	logger.Info(ctx, "Initializing game: %s", gameID)

	// create k8s cluster for this game session
	uc.gameEngine.InitializeGame(ctx, gameID, notifier)

	// TODO: Not implemented yet
	return nil
}

// StartGame creates initial resources for the cluster and begins the game session for the specified game ID.
// If user is already in a game, user joins the existing game. This is assumed to be idempotent.
// If the game is over, start a new game session.
func (uc *GameInteractor) StartGame(ctx context.Context, connID string) error {
	gameID := uc.getGameID(connID)
	logger.Info(ctx, "Starting game: %s", gameID)

	// call game engine to start the game
	return uc.gameEngine.StartGame(ctx, gameID)
}

// StopGame stops the game session for the specified game ID and and removes all associated resources.
func (uc *GameInteractor) CleanupGame(ctx context.Context, connID string) error {
	gameID := uc.getGameID(connID)
	logger.Info(ctx, "Stopping game: %s", gameID)

	// Stop game engine
	if err := uc.gameEngine.CleanupGame(ctx, gameID); err != nil {
		logger.Error(ctx, "Failed to stop game engine: %v", err)
	}

	logger.Info(ctx, "Game %s stopped successfully", gameID)
	return nil
}

// SchedulePodToNode assigns a specific pod to a specific node within the game session.
// This should be invoked by user actions.
func (uc *GameInteractor) SchedulePodToNode(ctx context.Context, podID, nodeID, connID string) error {
	gameID := uc.getGameID(connID)
	logger.Info(ctx, "Connection %s requesting to schedule pod %s to node %s", connID, podID, nodeID)

	if err := uc.gameEngine.SchedulePod(ctx, gameID, podID, nodeID); err != nil {
		return fmt.Errorf("failed to schedule pod %s to node %s in game %s: %v", podID, nodeID, gameID, err)
	}
	return nil
}

func (uc *GameInteractor) initializeGamePods(ctx context.Context, gameID string) error {
	// Note: Initial pods are created by GameSessionService.createInitialPods
	// This method is kept for interface compatibility
	logger.Debug(ctx, "Game pods initialized for %s", gameID)
	return nil
}

func (uc *GameInteractor) generateInitialPods(gameID string) []*entity.Pod {
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
