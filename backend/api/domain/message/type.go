package message

// Client-Server Message Types
const (
	MessageTypePing        = "ping"
	MessageTypePong        = "pong"
	MessageTypeStartGame   = "start_game"
	MessageTypeSchedulePod = "schedule_pod"
)

// Server-Client Event Types
const (
	EventTypeSessionReady = "session_ready"
	EventTypeGameStarted  = "game_started"
	EventTypeGameOver     = "game_over"
	EventTypeGameUpdate   = "game_update"
	EventTypePodCreated   = "pod_created"
	EventTypePodScheduled = "pod_scheduled"
	EventTypeError        = "error"
	EventTypeServerPing   = "server_ping"
)
