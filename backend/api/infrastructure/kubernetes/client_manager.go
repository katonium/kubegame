package kubernetes

import (
	"context"
	"fmt"
	"sync"

	"github.com/katonium/kubegame/backend/adapter/kubernetes"
)

// clusterManager is a simple in-memory implementation of ClusterManager
type clusterManager struct {
	clusters map[string]kubernetes.Cluster
	mutex    sync.RWMutex
}

// NewClusterManager creates a new fake cluster manager
func NewClusterManager() kubernetes.ClusterManager {
	return &clusterManager{
		clusters: make(map[string]kubernetes.Cluster),
	}
}

func (m *clusterManager) CreateCluster(ctx context.Context, ID string) (kubernetes.Cluster, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.clusters[ID]; exists {
		return nil, fmt.Errorf("cluster %s already exists", ID)
	}

	client := NewFakeClient()
	m.clusters[ID] = client
	return client, nil
}

func (m *clusterManager) GetCluster(ctx context.Context, ID string) (kubernetes.Cluster, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	client, exists := m.clusters[ID]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", ID)
	}

	return client, nil
}

func (m *clusterManager) DeleteCluster(ctx context.Context, ID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cluster, exists := m.clusters[ID]
	if !exists {
		return fmt.Errorf("cluster %s not found", ID)
	}

	if err := cluster.Close(ctx); err != nil {
		return fmt.Errorf("failed to close cluster %s: %w", ID, err)
	}

	delete(m.clusters, ID)
	return nil
}

func (m *clusterManager) ListClusters(ctx context.Context) (clusterIDs []string, err error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	clusterIDs = make([]string, 0, len(m.clusters))
	for clusterID := range m.clusters {
		clusterIDs = append(clusterIDs, clusterID)
	}

	return clusterIDs, nil
}
