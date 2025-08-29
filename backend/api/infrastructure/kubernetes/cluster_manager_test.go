package kubernetes

import (
	"context"
	"strings"
	"testing"

	"github.com/katonium/kubegame/backend/domain/entity"
)

func TestFakeClusterManager(t *testing.T) {
	ctx := context.Background()
	manager := NewFakeClusterManager()

	t.Run("CreateCluster", func(t *testing.T) {
		client, err := manager.CreateCluster(ctx, "cluster-1")
		if err != nil {
			t.Fatalf("Failed to create cluster: %v", err)
		}
		if client == nil {
			t.Fatal("Expected client to be non-nil")
		}

		// Verify cluster was created by listing
		clusters, err := manager.ListClusters(ctx)
		if err != nil {
			t.Fatalf("Failed to list clusters: %v", err)
		}
		if len(clusters) != 1 {
			t.Fatalf("Expected 1 cluster, got %d", len(clusters))
		}
		if clusters[0] != "cluster-1" {
			t.Errorf("Expected cluster ID 'cluster-1', got '%s'", clusters[0])
		}
	})

	t.Run("CreateCluster_Duplicate", func(t *testing.T) {
		_, err := manager.CreateCluster(ctx, "cluster-1") // Same ID as above
		if err == nil {
			t.Error("Expected error when creating duplicate cluster, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected error message to contain 'already exists', got: %v", err)
		}
	})

	t.Run("GetCluster", func(t *testing.T) {
		client, err := manager.GetCluster(ctx, "cluster-1")
		if err != nil {
			t.Fatalf("Failed to get cluster: %v", err)
		}
		if client == nil {
			t.Fatal("Expected client to be non-nil")
		}

		// Test that the client works by creating a node
		node := &entity.Node{
			ID:       "test-node",
			Name:     "Test Node",
			NodeType: "player",
			Capacity: entity.NodeCapacity{CPU: 2, Memory: 4},
		}
		err = client.CreateNode(ctx, "default", node)
		if err != nil {
			t.Fatalf("Failed to create node in retrieved cluster: %v", err)
		}
	})

	t.Run("GetCluster_NotFound", func(t *testing.T) {
		_, err := manager.GetCluster(ctx, "non-existent-cluster")
		if err == nil {
			t.Error("Expected error when getting non-existent cluster, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected error message to contain 'not found', got: %v", err)
		}
	})

	t.Run("ListClusters_Multiple", func(t *testing.T) {
		// Create another cluster
		_, err := manager.CreateCluster(ctx, "cluster-2")
		if err != nil {
			t.Fatalf("Failed to create second cluster: %v", err)
		}

		clusters, err := manager.ListClusters(ctx)
		if err != nil {
			t.Fatalf("Failed to list clusters: %v", err)
		}
		if len(clusters) != 2 {
			t.Fatalf("Expected 2 clusters, got %d", len(clusters))
		}

		// Verify both clusters exist
		found1, found2 := false, false
		for _, id := range clusters {
			if id == "cluster-1" {
				found1 = true
			}
			if id == "cluster-2" {
				found2 = true
			}
		}
		if !found1 || !found2 {
			t.Errorf("Expected to find both cluster-1 and cluster-2, found IDs: %v", clusters)
		}
	})

	t.Run("DeleteCluster", func(t *testing.T) {
		err := manager.DeleteCluster(ctx, "cluster-2")
		if err != nil {
			t.Fatalf("Failed to delete cluster: %v", err)
		}

		// Verify cluster was deleted
		clusters, err := manager.ListClusters(ctx)
		if err != nil {
			t.Fatalf("Failed to list clusters after deletion: %v", err)
		}
		if len(clusters) != 1 {
			t.Fatalf("Expected 1 cluster after deletion, got %d", len(clusters))
		}
		if clusters[0] != "cluster-1" {
			t.Errorf("Expected remaining cluster to be 'cluster-1', got '%s'", clusters[0])
		}
	})

	t.Run("DeleteCluster_NotFound", func(t *testing.T) {
		err := manager.DeleteCluster(ctx, "non-existent-cluster")
		if err == nil {
			t.Error("Expected error when deleting non-existent cluster, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected error message to contain 'not found', got: %v", err)
		}
	})
}

func TestFakeClusterManager_ClusterIsolation(t *testing.T) {
	ctx := context.Background()
	manager := NewFakeClusterManager()

	// Create two clusters
	client1, err := manager.CreateCluster(ctx, "cluster-1")
	if err != nil {
		t.Fatalf("Failed to create cluster-1: %v", err)
	}

	client2, err := manager.CreateCluster(ctx, "cluster-2")
	if err != nil {
		t.Fatalf("Failed to create cluster-2: %v", err)
	}

	// Create nodes in each cluster
	node1 := &entity.Node{
		ID:       "node-1",
		Name:     "Node in Cluster 1",
		NodeType: "player",
		Capacity: entity.NodeCapacity{CPU: 2, Memory: 4},
	}
	if err := client1.CreateNode(ctx, "default", node1); err != nil {
		t.Fatalf("Failed to create node in cluster-1: %v", err)
	}

	node2 := &entity.Node{
		ID:       "node-2",
		Name:     "Node in Cluster 2",
		NodeType: "cpu",
		Capacity: entity.NodeCapacity{CPU: 4, Memory: 8},
	}
	if err := client2.CreateNode(ctx, "default", node2); err != nil {
		t.Fatalf("Failed to create node in cluster-2: %v", err)
	}

	// Verify isolation: each cluster should only see its own nodes
	namespace := "default"
	nodes1, err := client1.GetNodes(ctx, &namespace)
	if err != nil {
		t.Fatalf("Failed to get nodes from cluster-1: %v", err)
	}
	if len(nodes1) != 1 {
		t.Fatalf("Expected cluster-1 to have 1 node, got %d", len(nodes1))
	}
	if nodes1[0].ID != "node-1" {
		t.Errorf("Expected cluster-1 to have node-1, got %s", nodes1[0].ID)
	}

	nodes2, err := client2.GetNodes(ctx, &namespace)
	if err != nil {
		t.Fatalf("Failed to get nodes from cluster-2: %v", err)
	}
	if len(nodes2) != 1 {
		t.Fatalf("Expected cluster-2 to have 1 node, got %d", len(nodes2))
	}
	if nodes2[0].ID != "node-2" {
		t.Errorf("Expected cluster-2 to have node-2, got %s", nodes2[0].ID)
	}
}
