package schema

import "time"

// BaseMessage represents the common structure of all WebSocket messages
type BaseMessage struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// Client-to-Server Messages

// PingMessage represents a ping from client to server
type PingMessage struct {
	BaseMessage
}

// PongMessage represents a pong response to server ping
type PongMessage struct {
	BaseMessage
}

// StartGameMessage represents a request to start a new game
type StartGameMessage struct {
	BaseMessage
}

// SchedulePodData represents the data for scheduling a pod
type SchedulePodData struct {
	PodID  string `json:"pod_id"`
	NodeID string `json:"node_id"`
}

// SchedulePodMessage represents a request to schedule a pod to a node
type SchedulePodMessage struct {
	BaseMessage
	Data SchedulePodData `json:"data"`
}

// Server-to-Client Events

// SessionReadyEvent represents session creation confirmation
type SessionReadyEvent struct {
	BaseMessage
}

// GameStartedEvent represents game start confirmation
type GameStartedEvent struct {
	BaseMessage
	Data GameInfo `json:"data"`
}

// GameUpdateEvent represents periodic game state update
type GameUpdateEvent struct {
	BaseMessage
	Data GameInfo `json:"data"`
}

// GameOverEvent represents game end event
type GameOverEvent struct {
	BaseMessage
	Data GameInfo `json:"data"`
}

// PodCreatedEvent represents new pod creation event
type PodCreatedEvent struct {
	BaseMessage
	Data GameInfo `json:"data"`
}

// PodScheduledEvent represents pod scheduling event
type PodScheduledEvent struct {
	BaseMessage
	Data GameInfo `json:"data"`
}

// ErrorData represents error information
type ErrorData struct {
	Message string                 `json:"message"`
	Code    ErrorCode              `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorEvent represents an error event
type ErrorEvent struct {
	BaseMessage
	Data ErrorData `json:"data"`
}

// ServerPingEvent represents a ping from server to client
type ServerPingEvent struct {
	BaseMessage
	Data interface{} `json:"data"` // null
}

// Message type constants
const (
	MessageTypePing        = "ping"
	MessageTypePong        = "pong"
	MessageTypeStartGame   = "start_game"
	MessageTypeSchedulePod = "schedule_pod"
	
	EventTypeSessionReady  = "session_ready"
	EventTypeGameStarted   = "game_started"
	EventTypeGameUpdate    = "game_update"
	EventTypeGameOver      = "game_over"
	EventTypePodCreated    = "pod_created"
	EventTypePodScheduled  = "pod_scheduled"
	EventTypeError         = "error"
	EventTypeServerPing    = "ping"
)
