package service

import (
	"context"

	"github.com/katonium/kubegame/backend/domain/entity"
)

type KubernetesService interface {
	// Node operations
	CreateNode(ctx context.Context, node *entity.Node) error
	GetNodes(ctx context.Context) ([]*entity.Node, error)
	DeleteNode(ctx context.Context, nodeID string) error

	// Pod operations
	CreatePod(ctx context.Context, pod *entity.Pod) error
	GetPods(ctx context.Context) ([]*entity.Pod, error)
	UpdatePodStatus(ctx context.Context, podID string, status entity.PodStatus) error
	DeletePod(ctx context.Context, podID string) error

	// Scheduling
	SchedulePod(ctx context.Context, podID string, nodeID string) error
	CanSchedulePod(ctx context.Context, pod *entity.Pod, node *entity.Node) (bool, error)
	GetNodeResourceUsage(ctx context.Context, nodeID string) (*entity.ResourceRequirements, error)
}

type SchedulerService interface {
	// Start the Kubernetes scheduler instance
	Start(ctx context.Context) error

	// Stop the Kubernetes scheduler instance
	Stop(ctx context.Context) error

	// Check scheduler health
	IsHealthy(ctx context.Context) bool

	// Watch for scheduling events
	WatchSchedulingEvents(ctx context.Context) (<-chan *entity.GameEvent, error)
}
