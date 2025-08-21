package schema

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageHandler provides utilities for handling WebSocket messages
type MessageHandler struct{}

// NewMessageHandler creates a new message handler
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{}
}

// CreatePingMessage creates a ping message
func (h *MessageHandler) CreatePingMessage() PingMessage {
	return PingMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypePing,
			Timestamp: time.Now(),
		},
		Data: nil,
	}
}

// CreatePongMessage creates a pong message
func (h *MessageHandler) CreatePongMessage() PongMessage {
	return PongMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypePong,
			Timestamp: time.Now(),
		},
		Data: nil,
	}
}

// CreateStartGameMessage creates a start game message
func (h *MessageHandler) CreateStartGameMessage() StartGameMessage {
	return StartGameMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeStartGame,
			Timestamp: time.Now(),
		},
		Data: nil,
	}
}

// CreateSchedulePodMessage creates a schedule pod message
func (h *MessageHandler) CreateSchedulePodMessage(podID, nodeID string) SchedulePodMessage {
	return SchedulePodMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeSchedulePod,
			Timestamp: time.Now(),
		},
		Data: SchedulePodData{
			PodID:  podID,
			NodeID: nodeID,
		},
	}
}

// CreateSessionReadyEvent creates a session ready event
func (h *MessageHandler) CreateSessionReadyEvent() SessionReadyEvent {
	return SessionReadyEvent{
		BaseMessage: BaseMessage{
			Type:      EventTypeSessionReady,
			Timestamp: time.Now(),
		},
		Data: nil,
	}
}

// CreateGameStartedEvent creates a game started event
func (h *MessageHandler) CreateGameStartedEvent(gameInfo GameInfo) GameStartedEvent {
	return GameStartedEvent{
		BaseMessage: BaseMessage{
			Type:      EventTypeGameStarted,
			Timestamp: time.Now(),
		},
		Data: gameInfo,
	}
}

// CreateGameUpdateEvent creates a game update event
func (h *MessageHandler) CreateGameUpdateEvent(gameInfo GameInfo) GameUpdateEvent {
	return GameUpdateEvent{
		BaseMessage: BaseMessage{
			Type:      EventTypeGameUpdate,
			Timestamp: time.Now(),
		},
		Data: gameInfo,
	}
}

// CreateGameOverEvent creates a game over event
func (h *MessageHandler) CreateGameOverEvent(playerScore, cpuScore int) GameOverEvent {
	return GameOverEvent{
		BaseMessage: BaseMessage{
			Type:      EventTypeGameOver,
			Timestamp: time.Now(),
		},
		Data: GameOverData{
			PlayerScore: playerScore,
			CPUScore:    cpuScore,
		},
	}
}

// CreatePodCreatedEvent creates a pod created event
func (h *MessageHandler) CreatePodCreatedEvent(gameInfo GameInfo) PodCreatedEvent {
	return PodCreatedEvent{
		BaseMessage: BaseMessage{
			Type:      EventTypePodCreated,
			Timestamp: time.Now(),
		},
		Data: gameInfo,
	}
}

// CreatePodScheduledEvent creates a pod scheduled event
func (h *MessageHandler) CreatePodScheduledEvent(gameInfo GameInfo) PodScheduledEvent {
	return PodScheduledEvent{
		BaseMessage: BaseMessage{
			Type:      EventTypePodScheduled,
			Timestamp: time.Now(),
		},
		Data: gameInfo,
	}
}

// CreateErrorEvent creates an error event
func (h *MessageHandler) CreateErrorEvent(message string, code *ErrorCode, details map[string]interface{}) ErrorEvent {
	return ErrorEvent{
		BaseMessage: BaseMessage{
			Type:      EventTypeError,
			Timestamp: time.Now(),
		},
		Data: ErrorData{
			Message: message,
			Code:    code,
			Details: details,
		},
	}
}

// CreateServerPingEvent creates a server ping event
func (h *MessageHandler) CreateServerPingEvent() ServerPingEvent {
	return ServerPingEvent{
		BaseMessage: BaseMessage{
			Type:      EventTypeServerPing,
			Timestamp: time.Now(),
		},
		Data: nil,
	}
}

// ParseMessage parses a raw JSON message and returns the message type
func (h *MessageHandler) ParseMessage(data []byte) (string, error) {
	var base BaseMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return "", fmt.Errorf("failed to parse message: %w", err)
	}
	return base.Type, nil
}

// ParseClientMessage parses a client message based on its type
func (h *MessageHandler) ParseClientMessage(data []byte, messageType string) (interface{}, error) {
	switch messageType {
	case MessageTypePing:
		var msg PingMessage
		err := json.Unmarshal(data, &msg)
		return msg, err
	case MessageTypePong:
		var msg PongMessage
		err := json.Unmarshal(data, &msg)
		return msg, err
	case MessageTypeStartGame:
		var msg StartGameMessage
		err := json.Unmarshal(data, &msg)
		return msg, err
	case MessageTypeSchedulePod:
		var msg SchedulePodMessage
		err := json.Unmarshal(data, &msg)
		return msg, err
	default:
		return nil, fmt.Errorf("unknown message type: %s", messageType)
	}
}
