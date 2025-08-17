package service

import (
	"context"
	"github.com/katonium/kubegame/backend/domain/entity"
)

type GameSessionService interface {
	// Session lifecycle management
	CreateSession(ctx context.Context, connectionID string) (*entity.GameSession, error)
	GetSession(ctx context.Context, connectionID string) (*entity.GameSession, error)
	UpdateSessionActivity(ctx context.Context, connectionID string) error
	CloseSession(ctx context.Context, connectionID string) error
	
	// Cluster management per session
	CreateCluster(ctx context.Context, connectionID string) error
	DeleteCluster(ctx context.Context, connectionID string) error
	
	// Game lifecycle per session
	StartGame(ctx context.Context, connectionID string) error
	StopGame(ctx context.Context, connectionID string) error
	RestartGame(ctx context.Context, connectionID string) error
	
	// Session state management
	GetSessionState(ctx context.Context, connectionID string) (*entity.GameSession, error)
	
	// Cleanup
	CleanupInactiveSessions(ctx context.Context) error
}