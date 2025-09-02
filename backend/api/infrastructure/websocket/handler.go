// Package websocket provides WebSocket multiplexing functionality for handling different message types.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/katonium/kubegame/backend/generated"
	"github.com/katonium/kubegame/backend/usecase"
	"github.com/katonium/kubegame/backend/util/logger"
)

type MessageHandler interface {
	OnConnect(ctx context.Context, cli Client) error
	OnDisconnect(ctx context.Context, clientID string) error
	HandleMessage(ctx context.Context, msg json.RawMessage, cli Client) error
}

// messageHandler handles WebSocket message multiplexing based on message type.
type messageHandler struct {
	gameInteractor *usecase.GameInteractor
}

func NewMessageHandler(gameInteractor *usecase.GameInteractor) MessageHandler {
	return &messageHandler{
		gameInteractor: gameInteractor,
	}
}

// OnConnect handles new client connections.
func (m *messageHandler) OnConnect(ctx context.Context, cli Client) error {
	logger.Info(ctx, "Client %s connected", cli.ID())

	// Create a k8s cluster for this connection
	m.gameInteractor.InitializeGame(ctx, cli.ID(), NewNotifier(cli))

	// Send welcome message
	welcomeResponse := &generated.SessionReadyEvent{
		Type:      "session_ready",
		Timestamp: time.Now(),
	}

	return cli.SendJSON(ctx, welcomeResponse)
}

// OnDisconnect handles client disconnections.
func (m *messageHandler) OnDisconnect(ctx context.Context, clientID string) error {
	logger.Info(ctx, "Client %s disconnected", clientID)
	// Cleanup resources
	if err := m.gameInteractor.CleanupGame(ctx, clientID); err != nil {
		logger.Error(ctx, "Failed to cleanup game for client %s: %v", clientID, err)
	}
	return nil
}

// HandleMessage processes incoming WebSocket JSON messages by reading the "type" field
// and unmarshaling to the appropriate struct before calling adapter functions.
func (m *messageHandler) HandleMessage(ctx context.Context, rawMessage json.RawMessage, cli Client) error {
	// First, extract the type field to determine message structure
	var baseMsg struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(rawMessage, &baseMsg); err != nil {
		return m.writeErrorResponse(ctx, cli, NewInvalidRequestError(fmt.Sprintf("failed to parse base message: %v", err)))
	}

	logger.Debug(ctx, "Processing message type '%s' from client %s", baseMsg.Type, cli.ID())

	// Define API rounte for each message type
	route := map[string]func(context.Context, Client, json.RawMessage) error{
		"ping": func(ctx context.Context, cli Client, msg json.RawMessage) error {
			var pingMsg generated.PingMessage
			if unmarshalErr := json.Unmarshal(msg, &pingMsg); unmarshalErr != nil {
				return NewInvalidRequestError(fmt.Sprintf("failed to unmarshal ping message: %v", unmarshalErr))
			}
			return m.handlePing(ctx, cli, &pingMsg)
		},
		"pong": func(ctx context.Context, cli Client, msg json.RawMessage) error {
			var pongMsg generated.PongMessage
			if unmarshalErr := json.Unmarshal(msg, &pongMsg); unmarshalErr != nil {
				return NewInvalidRequestError(fmt.Sprintf("failed to unmarshal pong message: %v", unmarshalErr))
			}
			return m.handlePong(ctx, cli, &pongMsg)
		},
		"start_game": func(ctx context.Context, cli Client, msg json.RawMessage) error {
			var startGameMsg generated.StartGameMessage
			if unmarshalErr := json.Unmarshal(msg, &startGameMsg); unmarshalErr != nil {
				return NewInvalidRequestError(fmt.Sprintf("failed to unmarshal start_game message: %v", unmarshalErr))
			}
			return m.handleStartGame(ctx, cli, &startGameMsg)
		},
		"schedule_pod": func(ctx context.Context, cli Client, msg json.RawMessage) error {
			var schedulePodMsg generated.SchedulePodMessage
			if unmarshalErr := json.Unmarshal(msg, &schedulePodMsg); unmarshalErr != nil {
				return NewInvalidRequestError(fmt.Sprintf("failed to unmarshal schedule_pod message: %v", unmarshalErr))
			}
			return m.handleSchedulePod(ctx, cli, &schedulePodMsg)
		},
	}

	// Route message based on type
	var err error
	if routeFunc, exists := route[baseMsg.Type]; exists {
		err = routeFunc(ctx, cli, rawMessage)
	} else {
		err = NewInvalidRequestError(fmt.Sprintf("unknown message type: %s", baseMsg.Type))
	}

	// Handle errors centrally - check if it's a GameError and send appropriate response
	if err != nil {
		if gameErr, ok := err.(*GameError); ok {
			return m.writeErrorResponse(ctx, cli, gameErr)
		}
		// For non-GameError errors, wrap as InvalidRequestError
		return m.writeErrorResponse(ctx, cli, NewInvalidRequestError(err.Error()))
	}

	return nil
}

// handlePing processes ping messages and returns a pong response.
func (m *messageHandler) handlePing(ctx context.Context, cli Client, msg *generated.PingMessage) error {
	logger.Debug(ctx, "Handling ping from client %s", cli.ID())

	response := &generated.PongMessage{
		Type:      "pong",
		Timestamp: time.Now(),
	}

	return cli.SendJSON(ctx, response)
}

// handlePong processes pong messages (no response needed).
func (m *messageHandler) handlePong(ctx context.Context, cli Client, msg *generated.PongMessage) error {
	logger.Debug(ctx, "Received pong from client %s", cli.ID())
	// No response needed for pong
	return nil
}

// handleStartGame processes start game messages.
func (m *messageHandler) handleStartGame(ctx context.Context, cli Client, msg *generated.StartGameMessage) error {
	logger.Info(ctx, "Handling start_game request from client %s", cli.ID())

	if m.gameInteractor == nil {
		return NewInvalidRequestError("Game use case not available")
	}

	// For now, assume gameID matches clientID - in a real implementation this would be different
	if err := m.gameInteractor.StartGame(ctx, cli.ID()); err != nil {
		logger.Error(ctx, "Failed to start game for client %s: %v", cli.ID(), err)
		return NewGameNotRunningError(fmt.Sprintf("Failed to start game: %v", err))
	}

	// Return game started event
	response := &generated.GameStartedEvent{
		Type:      "game_started",
		Timestamp: time.Now(),
		Data: generated.GameInfo{
			PlayerScore: 0,
			CpuScore:    0,
			TimeLeft:    300, // 5 minutes default
			PlayerCluster: generated.ClusterInfo{
				Nodes: []generated.NodeInfo{},
				Pods:  []generated.PodInfo{},
			},
			SchedulerCluster: generated.ClusterInfo{
				Nodes: []generated.NodeInfo{},
				Pods:  []generated.PodInfo{},
			},
		},
	}

	return cli.SendJSON(ctx, response)
}

// handleSchedulePod processes pod scheduling messages.
func (m *messageHandler) handleSchedulePod(ctx context.Context, cli Client, msg *generated.SchedulePodMessage) error {
	logger.Info(ctx, "Handling schedule_pod request from client %s: pod %s to node %s",
		cli.ID(), msg.Data.PodId, msg.Data.NodeId)

	if m.gameInteractor == nil {
		return NewInvalidRequestError("Game use case not available")
	}

	// Call the game use case to schedule the pod
	if err := m.gameInteractor.SchedulePodToNode(ctx, msg.Data.PodId, msg.Data.NodeId, cli.ID()); err != nil {
		logger.Error(ctx, "Failed to schedule pod %s to node %s for client %s: %v",
			msg.Data.PodId, msg.Data.NodeId, cli.ID(), err)

		return NewInvalidRequestError(err.Error())
	}

	// Return pod scheduled event
	response := &generated.PodScheduledEvent{
		Type:      "pod_scheduled",
		Timestamp: time.Now(),
		Data: generated.GameInfo{
			// This would be populated with actual game state in a real implementation
			PlayerScore: 0,
			CpuScore:    0,
			TimeLeft:    0,
			PlayerCluster: generated.ClusterInfo{
				Nodes: []generated.NodeInfo{},
				Pods:  []generated.PodInfo{},
			},
			SchedulerCluster: generated.ClusterInfo{
				Nodes: []generated.NodeInfo{},
				Pods:  []generated.PodInfo{},
			},
		},
	}

	return cli.SendJSON(ctx, response)
}

// writeErrorResponse creates and writes a standardized error response.
func (m *messageHandler) writeErrorResponse(ctx context.Context, cli Client, gameErr *GameError) error {
	errorResponse := &generated.ErrorEvent{
		Type:      "error",
		Timestamp: time.Now(),
		Data: struct {
			Code    generated.ErrorEventDataCode `json:"code"`
			Details *map[string]interface{}      `json:"details,omitempty"`
			Message string                       `json:"message"`
		}{
			Code:    gameErr.Code,
			Message: gameErr.Message,
			Details: &gameErr.Details,
		},
	}

	return cli.SendJSON(ctx, errorResponse)
}
