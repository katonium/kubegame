package kubernetes

import (
	"context"
	"strings"
	"testing"

	"github.com/katonium/kubegame/backend/domain/entity"
)

func TestFakeClient_NodeOperations(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient("test-cluster")

	t.Run("CreateNode", func(t *testing.T) {
		node := &entity.Node{
			ID:       "node-1",
			Name:     "Test Node 1",
			NodeType: "player",
			Capacity: entity.NodeCapacity{
				CPU:    4,
				Memory: 8,
			},
		}

		err := client.CreateNode(ctx, "default", node)
		if err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}

		// Verify node was created by listing
		namespace := "default"
		nodes, err := client.GetNodes(ctx, &namespace)
		if err != nil {
			t.Fatalf("Failed to get nodes: %v", err)
		}
		if len(nodes) != 1 {
			t.Fatalf("Expected 1 node, got %d", len(nodes))
		}
		if nodes[0].ID != "node-1" {
			t.Errorf("Expected node ID 'node-1', got '%s'", nodes[0].ID)
		}
		if nodes[0].Name != "Test Node 1" {
			t.Errorf("Expected node name 'Test Node 1', got '%s'", nodes[0].Name)
		}
		if nodes[0].NodeType != "player" {
			t.Errorf("Expected node type 'player', got '%s'", nodes[0].NodeType)
		}
		if nodes[0].Capacity.CPU != 4 {
			t.Errorf("Expected CPU capacity 4, got %d", nodes[0].Capacity.CPU)
		}
		if nodes[0].Capacity.Memory != 8 {
			t.Errorf("Expected memory capacity 8, got %d", nodes[0].Capacity.Memory)
		}
	})

	t.Run("CreateNode_Duplicate", func(t *testing.T) {
		node := &entity.Node{
			ID:       "node-1", // Same ID as above
			Name:     "Duplicate Node",
			NodeType: "cpu",
			Capacity: entity.NodeCapacity{CPU: 2, Memory: 4},
		}

		err := client.CreateNode(ctx, "default", node)
		if err == nil {
			t.Error("Expected error when creating duplicate node, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected error message to contain 'already exists', got: %v", err)
		}
	})
}

func TestFakeClient_PodOperations(t *testing.T) {
	client := NewFakeClient("test-cluster")
	ctx := context.Background()

	t.Run("CreatePod", func(t *testing.T) {
		pod := &entity.Pod{
			ID:    "pod-1",
			Name:  "Test Pod 1",
			Label: entity.PodLabelVanilla,
			Requirements: entity.ResourceRequirements{
				CPU:    1,
				Memory: 2,
			},
			Status:    entity.PodStatusPending,
			NodeID:    nil,
			Owner:     entity.PodOwnerPlayer,
			Namespace: "default",
		}

		err := client.CreatePod(ctx, "default", pod)
		if err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}

		// Verify pod was created by listing
		pods, err := client.GetPods(ctx, "default")
		if err != nil {
			t.Fatalf("Failed to get pods: %v", err)
		}
		if len(pods) != 1 {
			t.Fatalf("Expected 1 pod, got %d", len(pods))
		}
		if pods[0].ID != "pod-1" {
			t.Errorf("Expected pod ID 'pod-1', got '%s'", pods[0].ID)
		}
		if pods[0].Name != "Test Pod 1" {
			t.Errorf("Expected pod name 'Test Pod 1', got '%s'", pods[0].Name)
		}
		if pods[0].Label != entity.PodLabelVanilla {
			t.Errorf("Expected pod label %v, got %v", entity.PodLabelVanilla, pods[0].Label)
		}
		if pods[0].Status != entity.PodStatusPending {
			t.Errorf("Expected pod status %v, got %v", entity.PodStatusPending, pods[0].Status)
		}
		if pods[0].Requirements.CPU != 1 {
			t.Errorf("Expected CPU requirements 1, got %d", pods[0].Requirements.CPU)
		}
		if pods[0].Requirements.Memory != 2 {
			t.Errorf("Expected memory requirements 2, got %d", pods[0].Requirements.Memory)
		}
	})
}

func TestFakeClient_SchedulingOperations(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient("test-cluster")

	// Create a node
	node := &entity.Node{
		ID:       "node-1",
		Name:     "Test Node",
		NodeType: "player",
		Capacity: entity.NodeCapacity{CPU: 4, Memory: 8},
	}
	if err := client.CreateNode(ctx, "default", node); err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	// Create a pod
	pod := &entity.Pod{
		ID:    "pod-1",
		Name:  "Test Pod",
		Label: entity.PodLabelVanilla,
		Requirements: entity.ResourceRequirements{
			CPU:    2,
			Memory: 3,
		},
		Owner:     entity.PodOwnerPlayer,
		Namespace: "default",
	}
	if err := client.CreatePod(ctx, "default", pod); err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	t.Run("SchedulePod", func(t *testing.T) {
		err := client.SchedulePod(ctx, "default", "pod-1", "node-1")
		if err != nil {
			t.Fatalf("Failed to schedule pod: %v", err)
		}

		// Verify pod is now scheduled
		pods, err := client.GetPods(ctx, "default")
		if err != nil {
			t.Fatalf("Failed to get pods: %v", err)
		}
		if len(pods) != 1 {
			t.Fatalf("Expected 1 pod, got %d", len(pods))
		}
		if pods[0].NodeID == nil || *pods[0].NodeID != "node-1" {
			t.Errorf("Expected pod to be scheduled on node-1, got %v", pods[0].NodeID)
		}
		if pods[0].Status != entity.PodStatusRunning {
			t.Errorf("Expected pod status to be Running after scheduling, got %v", pods[0].Status)
		}
	})

	t.Run("GetNodeResourceUsage", func(t *testing.T) {
		usage, err := client.GetNodeResourceUsage(ctx, "default", "node-1")
		if err != nil {
			t.Fatalf("Failed to get node resource usage: %v", err)
		}
		if usage.CPU != 2 {
			t.Errorf("Expected CPU usage 2, got %d", usage.CPU)
		}
		if usage.Memory != 3 {
			t.Errorf("Expected memory usage 3, got %d", usage.Memory)
		}
	})
}
