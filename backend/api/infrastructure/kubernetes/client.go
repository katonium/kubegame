package kubernetes

import (
	"context"
	"fmt"

	adapter "github.com/katonium/kubegame/backend/adapter/kubernetes"
	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/util/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// fakeClient is a thin wrapper around Kubernetes fake client
// that implements the Client interface using actual Kubernetes API objects
type fakeClient struct {
	client kubernetes.Interface
}

// NewFakeClient creates a new fake Kubernetes client using k8s fake clientset
func NewFakeClient() adapter.Cluster {
	return &fakeClient{
		client: fake.NewSimpleClientset(),
	}
}

// CreateNamespace creates a new namespace in the cluster.
func (f *fakeClient) CreateNamespace(ctx context.Context, namespace string) error {
	_, err := f.client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"kubegame.io/namespace": namespace,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace in Kubernetes: %w", err)
	}
	return nil
}

// Node operations

func (f *fakeClient) CreateNode(ctx context.Context, namespace string, node *entity.Node) error {
	k8sNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: node.ID,
			Labels: map[string]string{
				"kubegame.io/node-id":    node.ID,
				"kubegame.io/node-name":  node.Name,
				"kubegame.io/namespace":  namespace,
				"kubernetes.io/hostname": node.ID,
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewQuantity(int64(node.Capacity.CPU), resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(int64(node.Capacity.Memory)*1024*1024*1024, resource.BinarySI),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewQuantity(int64(node.Capacity.CPU), resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(int64(node.Capacity.Memory)*1024*1024*1024, resource.BinarySI),
			},
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	_, err := f.client.CoreV1().Nodes().Create(ctx, k8sNode, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create node in Kubernetes: %w", err)
	}

	return nil
}

func (f *fakeClient) GetNodes(ctx context.Context, namespace string) ([]*entity.Node, error) {
	var labelSelector string
	if namespace != "" {
		labelSelector = "kubegame.io/namespace=" + namespace
	}

	nodeList, err := f.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes from Kubernetes: %w", err)
	}

	nodes := make([]*entity.Node, 0, len(nodeList.Items))
	for _, k8sNode := range nodeList.Items {
		node := f.convertK8sNodeToEntity(&k8sNode)
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (f *fakeClient) DeleteNode(ctx context.Context, namespace string, nodeID string) error {
	err := f.client.CoreV1().Nodes().Delete(ctx, nodeID, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete node from Kubernetes: %w", err)
	}

	return nil
}

// Pod operations

func (f *fakeClient) CreatePod(ctx context.Context, namespace string, pod *entity.Pod) error {
	k8sPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.ID,
			Namespace: namespace,
			Labels: map[string]string{
				"kubegame.io/pod-id":    pod.ID,
				"kubegame.io/pod-name":  pod.Name,
				"kubegame.io/pod-label": string(pod.Label),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "game-container",
					Image: "nginx:alpine", // Simple container for simulation
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(int64(pod.Requirements.CPU), resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity(int64(pod.Requirements.Memory)*1024*1024*1024, resource.BinarySI),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(int64(pod.Requirements.CPU), resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity(int64(pod.Requirements.Memory)*1024*1024*1024, resource.BinarySI),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	// Add nodeSelector if pod is being scheduled to a specific node
	if pod.NodeID != nil {
		k8sPod.Spec.NodeSelector = map[string]string{
			"kubegame.io/node-id": *pod.NodeID,
		}
	}

	_, err := f.client.CoreV1().Pods(namespace).Create(ctx, k8sPod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod in Kubernetes: %w", err)
	}

	return nil
}

func (f *fakeClient) GetPods(ctx context.Context, namespace string) ([]*entity.Pod, error) {
	podList, err := f.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods from Kubernetes: %w", err)
	}

	pods := make([]*entity.Pod, 0, len(podList.Items))
	for _, k8sPod := range podList.Items {
		pod := f.convertK8sPodToEntity(&k8sPod)
		pods = append(pods, pod)
	}

	return pods, nil
}

func (f *fakeClient) DeletePod(ctx context.Context, namespace string, podID string) error {
	err := f.client.CoreV1().Pods(namespace).Delete(ctx, podID, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod from Kubernetes: %w", err)
	}

	return nil
}

// Scheduling operations

func (f *fakeClient) SchedulePod(ctx context.Context, namespace string, podID string, nodeID string) error {
	k8sPod, err := f.client.CoreV1().Pods(namespace).Get(ctx, podID, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod from Kubernetes: %w", err)
	}

	// Update pod to be scheduled on the specified node
	k8sPod.Spec.NodeName = nodeID
	k8sPod.Status.Phase = corev1.PodRunning

	_, err = f.client.CoreV1().Pods(namespace).Update(ctx, k8sPod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to schedule pod in Kubernetes: %w", err)
	}

	return nil
}

func (f *fakeClient) GetNodeResourceUsage(ctx context.Context, namespace string, nodeID string) (*entity.ResourceRequirements, error) {
	// Get all pods scheduled on this node
	podList, err := f.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node %s: %w", nodeID, err)
	}

	totalCPU := 0
	totalMemory := 0

	for _, k8sPod := range podList.Items {
		if k8sPod.Status.Phase == corev1.PodRunning {
			for _, container := range k8sPod.Spec.Containers {
				if cpuReq := container.Resources.Requests.Cpu(); cpuReq != nil {
					totalCPU += int(cpuReq.Value())
				}
				if memReq := container.Resources.Requests.Memory(); memReq != nil {
					totalMemory += int(memReq.Value() / (1024 * 1024 * 1024))
				}
			}
		}
	}

	return &entity.ResourceRequirements{
		CPU:    totalCPU,
		Memory: totalMemory,
	}, nil
}

// Helper conversion methods

func (f *fakeClient) convertK8sNodeToEntity(k8sNode *corev1.Node) *entity.Node {
	cpuCapacity := int(k8sNode.Status.Capacity.Cpu().Value())
	memoryCapacity := int(k8sNode.Status.Capacity.Memory().Value() / (1024 * 1024 * 1024))

	nodeName := k8sNode.Name
	if nameLabel, exists := k8sNode.Labels["kubegame.io/node-name"]; exists {
		nodeName = nameLabel
	}

	namespace := ""
	if nsLabel, exists := k8sNode.Labels["kubegame.io/namespace"]; exists {
		namespace = nsLabel
	} else {
		logger.Error(context.Background(), "Node %s does not have namespace label", k8sNode.Name)
	}

	return &entity.Node{
		ID:   k8sNode.Name,
		Name: nodeName,
		Capacity: entity.NodeCapacity{
			CPU:    cpuCapacity,
			Memory: memoryCapacity,
		},
		Used: entity.NodeCapacity{
			// TODO
			CPU:    0,
			Memory: 0,
		},
		Namespace: namespace,
	}
}

func (f *fakeClient) convertK8sPodToEntity(k8sPod *corev1.Pod) *entity.Pod {
	// Extract resource requirements from first container
	cpuReq := 0
	memoryReq := 0
	if len(k8sPod.Spec.Containers) > 0 {
		container := k8sPod.Spec.Containers[0]
		if cpuRes := container.Resources.Requests.Cpu(); cpuRes != nil {
			cpuReq = int(cpuRes.Value())
		}
		if memRes := container.Resources.Requests.Memory(); memRes != nil {
			memoryReq = int(memRes.Value() / (1024 * 1024 * 1024))
		}
	}

	// Convert Kubernetes phase to entity status
	status := entity.PodStatusPending
	switch k8sPod.Status.Phase {
	case corev1.PodRunning:
		status = entity.PodStatusRunning
	case corev1.PodFailed:
		status = entity.PodStatusFailed
	case corev1.PodSucceeded:
		status = entity.PodStatusTerminated
	}

	// Extract labels
	podName := k8sPod.Name
	if nameLabel, exists := k8sPod.Labels["kubegame.io/pod-name"]; exists {
		podName = nameLabel
	}

	podLabel := entity.PodLabelVanilla // default
	if labelValue, exists := k8sPod.Labels["kubegame.io/pod-label"]; exists {
		podLabel = entity.PodLabel(labelValue)
	}

	var nodeID *string
	if k8sPod.Spec.NodeName != "" {
		nodeID = &k8sPod.Spec.NodeName
	}

	// Extract namespace from labels if exists
	var namespace string
	if nsLabel, exists := k8sPod.Labels["kubegame.io/namespace"]; exists {
		namespace = nsLabel
	} else {
		logger.Error(context.Background(), "Pod %s does not have namespace label", k8sPod.Name)
	}

	return &entity.Pod{
		ID:     k8sPod.Name,
		Name:   podName,
		Label:  podLabel,
		Status: status,
		NodeID: nodeID,
		Requirements: entity.ResourceRequirements{
			CPU:    cpuReq,
			Memory: memoryReq,
		},
		Namespace: namespace,
		CreatedAt: k8sPod.CreationTimestamp.Time,
	}
}

// Close closes the client and releases any resources.
func (f *fakeClient) Close(ctx context.Context) error {
	return nil
}
