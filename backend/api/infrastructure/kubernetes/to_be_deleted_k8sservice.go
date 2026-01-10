package kubernetes

import (
	"context"
	"fmt"

	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/service"
	"github.com/katonium/kubegame/backend/util/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type kubernetesService struct {
	client kubernetes.Interface
}

func NewKubernetesService() service.KubernetesService {
	// Create fake Kubernetes client for game simulation
	client := fake.NewSimpleClientset()

	return &kubernetesService{
		client: client,
	}
}

func (k *kubernetesService) CreateNode(ctx context.Context, node *entity.Node) error {
	logger.Debug(ctx, "Creating node %s in Kubernetes cluster", node.ID)

	k8sNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: node.ID,
			Labels: map[string]string{
				// "node-type":              node.NodeType,
				"kubegame.io/node-id":    node.ID,
				"kubegame.io/node-name":  node.Name,
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

	_, err := k.client.CoreV1().Nodes().Create(ctx, k8sNode, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create node in Kubernetes: %w", err)
	}

	logger.Info(ctx, "Created node %s (%s) with capacity %d CPU, %d GB memory",
		node.ID, node.Name, node.Capacity.CPU, node.Capacity.Memory)

	return nil
}

func (k *kubernetesService) GetNodes(ctx context.Context) ([]*entity.Node, error) {
	nodeList, err := k.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes from Kubernetes: %w", err)
	}

	nodes := make([]*entity.Node, 0, len(nodeList.Items))
	for _, k8sNode := range nodeList.Items {
		node := k.convertK8sNodeToEntity(&k8sNode)
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (k *kubernetesService) DeleteNode(ctx context.Context, nodeID string) error {
	err := k.client.CoreV1().Nodes().Delete(ctx, nodeID, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete node from Kubernetes: %w", err)
	}

	logger.Info(ctx, "Deleted node %s from Kubernetes cluster", nodeID)
	return nil
}

func (k *kubernetesService) CreatePod(ctx context.Context, pod *entity.Pod) error {
	logger.Debug(ctx, "Creating pod %s in Kubernetes cluster", pod.ID)

	k8sPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.ID,
			Namespace: "default",
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

	_, err := k.client.CoreV1().Pods("default").Create(ctx, k8sPod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod in Kubernetes: %w", err)
	}

	logger.Info(ctx, "Created pod %s (%s) requiring %d CPU, %d GB memory",
		pod.ID, pod.Name, pod.Requirements.CPU, pod.Requirements.Memory)

	return nil
}

func (k *kubernetesService) GetPods(ctx context.Context) ([]*entity.Pod, error) {
	podList, err := k.client.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods from Kubernetes: %w", err)
	}

	pods := make([]*entity.Pod, 0, len(podList.Items))
	for _, k8sPod := range podList.Items {
		pod := k.convertK8sPodToEntity(&k8sPod)
		pods = append(pods, pod)
	}

	return pods, nil
}

func (k *kubernetesService) UpdatePodStatus(ctx context.Context, podID string, status entity.PodStatus) error {
	k8sPod, err := k.client.CoreV1().Pods("default").Get(ctx, podID, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod from Kubernetes: %w", err)
	}

	// Update pod phase based on entity status
	switch status {
	case entity.PodStatusPending:
		k8sPod.Status.Phase = corev1.PodPending
	case entity.PodStatusRunning:
		k8sPod.Status.Phase = corev1.PodRunning
	case entity.PodStatusFailed:
		k8sPod.Status.Phase = corev1.PodFailed
	case entity.PodStatusTerminated:
		k8sPod.Status.Phase = corev1.PodSucceeded
	}

	_, err = k.client.CoreV1().Pods("default").UpdateStatus(ctx, k8sPod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update pod status in Kubernetes: %w", err)
	}

	return nil
}

func (k *kubernetesService) DeletePod(ctx context.Context, podID string) error {
	err := k.client.CoreV1().Pods("default").Delete(ctx, podID, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod from Kubernetes: %w", err)
	}

	logger.Info(ctx, "Deleted pod %s from Kubernetes cluster", podID)
	return nil
}

func (k *kubernetesService) SchedulePod(ctx context.Context, podID string, nodeID string) error {
	logger.Debug(ctx, "Scheduling pod %s to node %s", podID, nodeID)

	k8sPod, err := k.client.CoreV1().Pods("default").Get(ctx, podID, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod from Kubernetes: %w", err)
	}

	// Update pod to be scheduled on the specified node
	k8sPod.Spec.NodeName = nodeID
	k8sPod.Status.Phase = corev1.PodRunning

	_, err = k.client.CoreV1().Pods("default").Update(ctx, k8sPod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to schedule pod in Kubernetes: %w", err)
	}

	logger.Info(ctx, "Scheduled pod %s to node %s", podID, nodeID)
	return nil
}

func (k *kubernetesService) CanSchedulePod(ctx context.Context, pod *entity.Pod, node *entity.Node) (bool, error) {
	usage, err := k.GetNodeResourceUsage(ctx, node.ID)
	if err != nil {
		return false, err
	}

	availableCPU := node.Capacity.CPU - usage.CPU
	availableMemory := node.Capacity.Memory - usage.Memory

	canSchedule := availableCPU >= pod.Requirements.CPU && availableMemory >= pod.Requirements.Memory

	logger.Debug(ctx, "Node %s availability: CPU %d/%d, Memory %d/%d, can schedule: %v",
		node.ID, availableCPU, node.Capacity.CPU, availableMemory, node.Capacity.Memory, canSchedule)

	return canSchedule, nil
}

func (k *kubernetesService) GetNodeResourceUsage(ctx context.Context, nodeID string) (*entity.ResourceRequirements, error) {
	// Get all pods scheduled on this node
	podList, err := k.client.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
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

func (k *kubernetesService) convertK8sNodeToEntity(k8sNode *corev1.Node) *entity.Node {
	cpuCapacity := int(k8sNode.Status.Capacity.Cpu().Value())
	memoryCapacity := int(k8sNode.Status.Capacity.Memory().Value() / (1024 * 1024 * 1024))

	// nodeType := "cpu" // default
	// if nodeTypeLabel, exists := k8sNode.Labels["node-type"]; exists {
	// 	nodeType = nodeTypeLabel
	// }

	nodeName := k8sNode.Name
	if nameLabel, exists := k8sNode.Labels["kubegame.io/node-name"]; exists {
		nodeName = nameLabel
	}

	return &entity.Node{
		ID:   k8sNode.Name,
		Name: nodeName,
		// NodeType: nodeType,
		Capacity: entity.NodeCapacity{
			CPU:    cpuCapacity,
			Memory: memoryCapacity,
		},
	}
}

func (k *kubernetesService) convertK8sPodToEntity(k8sPod *corev1.Pod) *entity.Pod {
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
		CreatedAt: k8sPod.CreationTimestamp.Time,
	}
}
