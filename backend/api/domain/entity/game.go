package entity

import (
	"time"
)

type PodStatus string

const (
	PodStatusPending    PodStatus = "Pending"
	PodStatusScheduling PodStatus = "Scheduling"
	PodStatusRunning    PodStatus = "Running"
	PodStatusTerminated PodStatus = "Terminated"
	PodStatusFailed     PodStatus = "Failed"
)

type PodOwner string

const (
	PodOwnerPlayer PodOwner = "player"
	PodOwnerCPU    PodOwner = "cpu"
	PodOwnerNone   PodOwner = ""
)

type PodLabel string

const (
	PodLabelBanana     PodLabel = "Banana"
	PodLabelChocolate  PodLabel = "Chocolate"
	PodLabelStrawberry PodLabel = "Strawberry"
	PodLabelVanilla    PodLabel = "Vanilla"
)

type ResourceRequirements struct {
	CPU    int `json:"cpu"`    // in cores
	Memory int `json:"memory"` // in GB
}

type Pod struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Label        PodLabel             `json:"label"`
	Requirements ResourceRequirements `json:"requirements"`
	Status       PodStatus            `json:"status"`
	NodeID       *string              `json:"nodeId"`
	Owner        PodOwner             `json:"owner"`
	Namespace    string               `json:"namespace"` // Namespace where the pod is created
	CreatedAt    time.Time            `json:"createdAt"`
}

type NodeCapacity struct {
	CPU    int `json:"cpu"`
	Memory int `json:"memory"`
}

type Node struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Capacity NodeCapacity `json:"capacity"`
	NodeType string       `json:"nodeType"` // "player" or "cpu"
}

type GameState string

const (
	GameStatePreGame  GameState = "pre-game"
	GameStatePlaying  GameState = "playing"
	GameStateGameOver GameState = "game-over"
)

type Game struct {
	ID          string    `json:"id"`
	State       GameState `json:"state"`
	PlayerScore int       `json:"playerScore"`
	CPUScore    int       `json:"cpuScore"`
	TimeLeft    int       `json:"timeLeft"` // in seconds
	StartedAt   time.Time `json:"startedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type GameEvent struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type SessionState string

const (
	SessionStateConnected    SessionState = "connected"
	SessionStateClusterReady SessionState = "cluster-ready"
	SessionStatePlaying      SessionState = "playing"
	SessionStateDisconnected SessionState = "disconnected"
)

// GameSession represents a user's game session tied to their WebSocket connection
type GameSession struct {
	ConnectionID      string       `json:"connectionId"`
	SessionID         string       `json:"sessionId"`
	State             SessionState `json:"state"`
	ClusterID         string       `json:"clusterId"`         // Unique cluster identifier for this session
	PlayerNamespace   string       `json:"playerNamespace"`   // Namespace for player pods
	SchedulerNamespace string      `json:"schedulerNamespace"` // Namespace for k8s scheduler pods
	GameID            *string      `json:"gameId"`            // Current game ID if playing
	CreatedAt         time.Time    `json:"createdAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
	LastActivity      time.Time    `json:"lastActivity"`
}
