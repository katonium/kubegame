package kubernetes

import (
	"context"

	"github.com/katonium/kubegame/backend/domain/entity"
)

const (
	AllNamespaces = ""
)

// Cluster is an interface for interacting with a Kubernetes cluster.
type Cluster interface {

	// CreateNamespace creates a new namespace in the cluster.
	CreateNamespace(ctx context.Context, namespace string) error

	// Node operations
	// CreateNode creates a new node in the specified namespace.
	CreateNode(ctx context.Context, namespace string, node *entity.Node) error
	// GetNodes lists all nodes in the specified namespace. If namespace is empty string, lists nodes across all namespaces.
	GetNodes(ctx context.Context, namespace string) ([]*entity.Node, error)
	// DeleteNode deletes a node by its ID in the specified namespace.
	DeleteNode(ctx context.Context, namespace string, nodeID string) error

	// Pod operations
	// CreatePod creates a new pod in the specified namespace.
	CreatePod(ctx context.Context, namespace string, pod *entity.Pod) error
	// GetPods lists all pods in the specified namespace. If namespace is nil, lists pods across all namespaces.
	GetPods(ctx context.Context, namespace string) ([]*entity.Pod, error)
	// UpdatePodStatus updates the status of a pod by its ID in the specified namespace.
	DeletePod(ctx context.Context, namespace string, podID string) error

	// Scheduling
	// SchedulePod assigns a pod to a node in the specified namespace.
	SchedulePod(ctx context.Context, namespace string, podID string, nodeID string) error
	// GetNodeResourceUsage retrieves the current resource usage of a node in the specified namespace.
	GetNodeResourceUsage(ctx context.Context, namespace string, nodeID string) (*entity.ResourceRequirements, error)

	// Close closes the client and releases any resources.
	Close(ctx context.Context) error
}

// ClusterManager manages multiple Kubernetes clusters by its ID.
type ClusterManager interface {
	// CreateCluster creates a new Kubernetes cluster, returning a Client to interact with it.
	CreateCluster(ctx context.Context, ID string) (Cluster, error)

	// GetCluster retrieves an existing Kubernetes cluster by its ID.
	GetCluster(ctx context.Context, ID string) (Cluster, error)

	// DeleteCluster deletes a Kubernetes cluster by its ID.
	DeleteCluster(ctx context.Context, ID string) error

	// ListClusters lists all existing Kubernetes clusters.
	ListClusters(ctx context.Context) ([]string, error)
}
