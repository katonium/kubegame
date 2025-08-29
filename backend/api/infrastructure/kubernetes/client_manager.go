package kubernetes

import (
	"context"
	"fmt"
	"sync"

	"github.com/katonium/kubegame/backend/adapter/kubernetes"
)

// fakeClusterManager manages multiple fake Kubernetes clusters
type fakeClusterManager struct {
	clusters map[string]kubernetes.Client
	mutex    sync.RWMutex
}

// NewFakeClusterManager creates a new fake cluster manager
func NewFakeClusterManager() kubernetes.ClusterManager {
	return &fakeClusterManager{
		clusters: make(map[string]kubernetes.Client),
	}
}

func (m *fakeClusterManager) CreateCluster(ctx context.Context, ID string) (kubernetes.Client, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.clusters[ID]; exists {
		return nil, fmt.Errorf("cluster %s already exists", ID)
	}

	client := NewFakeClient(ID)
	m.clusters[ID] = client
	return client, nil
}

func (m *fakeClusterManager) GetCluster(ctx context.Context, ID string) (kubernetes.Client, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	client, exists := m.clusters[ID]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", ID)
	}

	return client, nil
}

func (m *fakeClusterManager) DeleteCluster(ctx context.Context, ID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.clusters[ID]; !exists {
		return fmt.Errorf("cluster %s not found", ID)
	}

	delete(m.clusters, ID)
	return nil
}

func (m *fakeClusterManager) ListClusters(ctx context.Context) (clusterIDs []string, err error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	clusterIDs = make([]string, 0, len(m.clusters))
	for clusterID := range m.clusters {
		clusterIDs = append(clusterIDs, clusterID)
	}

	return clusterIDs, nil
}
