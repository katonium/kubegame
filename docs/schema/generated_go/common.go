// Package schema contains generated types for KubeGame WebSocket API
package schema

import "time"

// PodLabel represents the type of pod
type PodLabel string

const (
	PodLabelBanana     PodLabel = "banana"
	PodLabelChocolate  PodLabel = "chocolate"
	PodLabelStrawberry PodLabel = "strawberry"
	PodLabelVanilla    PodLabel = "vanilla"
)

// PodStatus represents the status of a pod
type PodStatus string

const (
	PodStatusPending      PodStatus = "pending"
	PodStatusScheduling   PodStatus = "scheduling"
	PodStatusScheduled    PodStatus = "scheduled"
	PodStatusUnscheduled  PodStatus = "unscheduled"
	PodStatusTerminating  PodStatus = "terminating"
)

// NodeType represents the type of node
type NodeType string

const (
	NodeTypePlayer    NodeType = "player"
	NodeTypeScheduler NodeType = "scheduler"
)

// ErrorCode represents error codes for API responses
type ErrorCode string

const (
	ErrorCodePodNotFound           ErrorCode = "POD_NOT_FOUND"
	ErrorCodeNodeNotFound          ErrorCode = "NODE_NOT_FOUND"
	ErrorCodeInsufficientResources ErrorCode = "INSUFFICIENT_RESOURCES"
	ErrorCodeInvalidSession        ErrorCode = "INVALID_SESSION"
	ErrorCodeGameNotRunning        ErrorCode = "GAME_NOT_RUNNING"
	ErrorCodeClusterNotReady       ErrorCode = "CLUSTER_NOT_READY"
	ErrorCodeCrossSessionAccess    ErrorCode = "CROSS_SESSION_ACCESS"
	ErrorCodeInvalidRequest        ErrorCode = "INVALID_REQUEST"
)

// ResourceRequirements represents CPU and memory requirements
type ResourceRequirements struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

// AffinityRule represents a single affinity rule
type AffinityRule struct {
	Values []PodLabel `json:"values"`
}

// Affinity represents pod affinity and anti-affinity rules
type Affinity struct {
	PodAffinity         AffinityRule `json:"podAffinity"`
	PodAntiAffinity     AffinityRule `json:"podAntiAffinity"`
	NodeAffinity        AffinityRule `json:"nodeAffinity"`
	NodeAntiAffinity    AffinityRule `json:"nodeAntiAffinity"`
}

// Pod represents a Kubernetes pod in the game
type Pod struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Label        PodLabel              `json:"label"`
	Affinity     Affinity              `json:"affinity"`
	Requirements ResourceRequirements  `json:"requirements"`
	Status       PodStatus             `json:"status"`
	NodeID       *string               `json:"nodeID"`
}

// Node represents a Kubernetes node in the game
type Node struct {
	ID       string                `json:"id"`
	Name     string                `json:"name"`
	Type     NodeType              `json:"type"`
	Capacity ResourceRequirements  `json:"capacity"`
	Used     ResourceRequirements  `json:"used"`
}

// ClusterInfo represents information about a cluster
type ClusterInfo struct {
	Pods  []Pod  `json:"pods"`
	Nodes []Node `json:"nodes"`
}

// GameInfo represents complete game state information
type GameInfo struct {
	TimeLeft         int         `json:"timeLeft"`
	PlayerScore      int         `json:"playerScore"`
	CPUScore         int         `json:"cpuScore"`
	PlayerCluster    ClusterInfo `json:"playerCluster"`
	SchedulerCluster ClusterInfo `json:"schedulerCluster"`
}
