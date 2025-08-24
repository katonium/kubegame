// Package websocket provides WebSocket multiplexing functionality for handling different message types.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/katonium/kubegame/backend/generated"
	"github.com/katonium/kubegame/backend/util/logger"
)

type MessageHandler interface {
	OnConnect(ctx context.Context, clientID string, responseWriter io.Writer) error
	OnDisconnect(ctx context.Context, clientID string) error
	HandleMessage(ctx context.Context, clientID string, rawMessage json.RawMessage, responseWriter io.Writer) error
}

// MessageMux handles WebSocket message multiplexing based on message type.
type MessageMux struct {
	gameUseCase GameUseCase
}

// NewMessageMux creates a new message multiplexer.
func NewMessageMux(gameUseCase GameUseCase) MessageHandler {
	return &MessageMux{
		gameUseCase: gameUseCase,
	}
}

// OnConnect handles new client connections.
func (m *MessageMux) OnConnect(ctx context.Context, clientID string, responseWriter io.Writer) error {
	logger.Info(ctx, "Client %s connected", clientID)

	// Send welcome message
	welcomeResponse := &generated.SessionReadyEvent{
		Type:      "session_ready",
		Timestamp: time.Now(),
	}

	return m.writeResponse(responseWriter, welcomeResponse)
}

// OnDisconnect handles client disconnections.
func (m *MessageMux) OnDisconnect(ctx context.Context, clientID string) error {
	logger.Info(ctx, "Client %s disconnected", clientID)
	// Cleanup logic can be added here
	return nil
}

// HandleMessage processes incoming WebSocket JSON messages by reading the "type" field
// and unmarshaling to the appropriate struct before calling adapter functions.
func (m *MessageMux) HandleMessage(ctx context.Context, clientID string, rawMessage json.RawMessage, responseWriter io.Writer) error {
	// First, extract the type field to determine message structure
	var baseMsg struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(rawMessage, &baseMsg); err != nil {
		return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, fmt.Sprintf("failed to unmarshal base message: %v", err))
	}

	logger.Debug(ctx, "Processing message type '%s' from client %s", baseMsg.Type, clientID)

	// Route message based on type and unmarshal to appropriate struct
	switch baseMsg.Type {
	case "ping":
		var msg generated.PingMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, fmt.Sprintf("failed to unmarshal ping message: %v", err))
		}
		return m.handlePing(ctx, clientID, &msg, responseWriter)

	case "pong":
		var msg generated.PongMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, fmt.Sprintf("failed to unmarshal pong message: %v", err))
		}
		return m.handlePong(ctx, clientID, &msg, responseWriter)

	case "start_game":
		var msg generated.StartGameMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, fmt.Sprintf("failed to unmarshal start_game message: %v", err))
		}
		return m.handleStartGame(ctx, clientID, &msg, responseWriter)

	case "schedule_pod":
		var msg generated.SchedulePodMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, fmt.Sprintf("failed to unmarshal schedule_pod message: %v", err))
		}
		return m.handleSchedulePod(ctx, clientID, &msg, responseWriter)

	default:
		return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, fmt.Sprintf("unknown message type: %s", baseMsg.Type))
	}
}

// handlePing processes ping messages and returns a pong response.
func (m *MessageMux) handlePing(ctx context.Context, clientID string, msg *generated.PingMessage, responseWriter io.Writer) error {
	logger.Debug(ctx, "Handling ping from client %s", clientID)

	response := &generated.PongMessage{
		Type:      "pong",
		Timestamp: time.Now(),
	}

	return m.writeResponse(responseWriter, response)
}

// handlePong processes pong messages (no response needed).
func (m *MessageMux) handlePong(ctx context.Context, clientID string, msg *generated.PongMessage, responseWriter io.Writer) error {
	logger.Debug(ctx, "Received pong from client %s", clientID)
	// No response needed for pong
	return nil
}

// handleStartGame processes start game messages.
func (m *MessageMux) handleStartGame(ctx context.Context, clientID string, msg *generated.StartGameMessage, responseWriter io.Writer) error {
	logger.Info(ctx, "Handling start_game request from client %s", clientID)

	if m.gameUseCase == nil {
		return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, "Game use case not available")
	}

	// For now, assume gameID matches clientID - in a real implementation this would be different
	if err := m.gameUseCase.StartGame(ctx, clientID); err != nil {
		logger.Error(ctx, "Failed to start game for client %s: %v", clientID, err)
		return m.writeErrorResponse(responseWriter, generated.GAMENOTRUNNING, fmt.Sprintf("Failed to start game: %v", err))
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

	return m.writeResponse(responseWriter, response)
}

// handleSchedulePod processes pod scheduling messages.
func (m *MessageMux) handleSchedulePod(ctx context.Context, clientID string, msg *generated.SchedulePodMessage, responseWriter io.Writer) error {
	logger.Info(ctx, "Handling schedule_pod request from client %s: pod %s to node %s",
		clientID, msg.Data.PodId, msg.Data.NodeId)

	if m.gameUseCase == nil {
		return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, "Game use case not available")
	}

	// Call the game use case to schedule the pod
	if err := m.gameUseCase.SchedulePodToNode(ctx, msg.Data.PodId, msg.Data.NodeId, clientID); err != nil {
		logger.Error(ctx, "Failed to schedule pod %s to node %s for client %s: %v",
			msg.Data.PodId, msg.Data.NodeId, clientID, err)

		return m.writeErrorResponse(responseWriter, generated.INVALIDREQUEST, err.Error())
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

	return m.writeResponse(responseWriter, response)
}

// writeResponse writes a response object as JSON to the response writer.
func (m *MessageMux) writeResponse(responseWriter io.Writer, response interface{}) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	_, err = responseWriter.Write(data)
	return err
}

// writeErrorResponse creates and writes a standardized error response.
func (m *MessageMux) writeErrorResponse(responseWriter io.Writer, code generated.ErrorEventDataCode, message string) error {
	errorResponse := &generated.ErrorEvent{
		Type:      "error",
		Timestamp: time.Now(),
		Data: struct {
			Code    generated.ErrorEventDataCode `json:"code"`
			Details *map[string]interface{}      `json:"details,omitempty"`
			Message string                       `json:"message"`
		}{
			Code:    code,
			Message: message,
			Details: nil,
		},
	}

	return m.writeResponse(responseWriter, errorResponse)
}
