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
