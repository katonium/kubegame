package service

import (
	"context"
	"github.com/katonium/kubegame/backend/domain/entity"
	"net/http"
)

type WebSocketService interface {
	// HTTP handler for WebSocket upgrade
	http.Handler

	// Connection management
	RegisterClient(ctx context.Context, clientID string, conn interface{}) error
	UnregisterClient(ctx context.Context, clientID string) error
	GetConnectedClients(ctx context.Context) ([]string, error)

	// Message broadcasting
	BroadcastEvent(ctx context.Context, event *entity.GameEvent) error
	SendEventToClient(ctx context.Context, clientID string, event *entity.GameEvent) error

	// Game state broadcasting
	BroadcastGameState(ctx context.Context, gameState *entity.Game) error
	BroadcastPodUpdate(ctx context.Context, pod *entity.Pod) error
	BroadcastNodeUpdate(ctx context.Context, node *entity.Node) error
}
