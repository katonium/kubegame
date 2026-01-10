package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/service"
	"github.com/katonium/kubegame/backend/util/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type schedulerService struct {
	client          kubernetes.Interface
	eventChannel    chan *entity.GameEvent
	stopChannel     chan struct{}
	isRunning       bool
	informerFactory informers.SharedInformerFactory
}

func NewSchedulerService(client kubernetes.Interface) service.SchedulerService {
	return &schedulerService{
		client:       client,
		eventChannel: make(chan *entity.GameEvent, 100),
		stopChannel:  make(chan struct{}),
		isRunning:    false,
	}
}

func (s *schedulerService) Start(ctx context.Context) error {
	logger.Info(ctx, "Starting Kubernetes scheduler service")

	if s.isRunning {
		return fmt.Errorf("scheduler service is already running")
	}

	// Create shared informer factory
	s.informerFactory = informers.NewSharedInformerFactory(s.client, time.Second*30)

	// Set up pod informer to watch for scheduling events
	podInformer := s.informerFactory.Core().V1().Pods()

	// Add event handlers for pod events
	_, err := podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				s.handlePodEvent(ctx, "pod_created", pod)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if pod, ok := newObj.(*corev1.Pod); ok {
				s.handlePodEvent(ctx, "pod_updated", pod)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				s.handlePodEvent(ctx, "pod_deleted", pod)
			}
		},
	})

	if err != nil {
		return fmt.Errorf("failed to add pod event handler: %w", err)
	}

	// Start the informer
	s.informerFactory.Start(s.stopChannel)

	// Wait for cache to sync
	if !cache.WaitForCacheSync(s.stopChannel, podInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync pod informer cache")
	}

	s.isRunning = true
	logger.Info(ctx, "Kubernetes scheduler service started successfully")

	return nil
}

func (s *schedulerService) Stop(ctx context.Context) error {
	logger.Info(ctx, "Stopping Kubernetes scheduler service")

	if !s.isRunning {
		return fmt.Errorf("scheduler service is not running")
	}

	// Stop the informer
	close(s.stopChannel)
	s.isRunning = false

	logger.Info(ctx, "Kubernetes scheduler service stopped")
	return nil
}

func (s *schedulerService) IsHealthy(ctx context.Context) bool {
	return s.isRunning
}

func (s *schedulerService) WatchSchedulingEvents(ctx context.Context) (<-chan *entity.GameEvent, error) {
	if !s.isRunning {
		return nil, fmt.Errorf("scheduler service is not running")
	}

	return s.eventChannel, nil
}

func (s *schedulerService) handlePodEvent(ctx context.Context, eventType string, pod *corev1.Pod) {
	logger.Debug(ctx, "Handling pod event: %s for pod %s", eventType, pod.Name)

	// Convert Kubernetes pod to game entity
	gamePod := s.convertK8sPodToEntity(pod)

	// Create game event
	event := &entity.GameEvent{
		Type:      eventType,
		Data:      gamePod,
		Timestamp: time.Now(),
	}

	// Send event to channel (non-blocking)
	select {
	case s.eventChannel <- event:
		logger.Debug(ctx, "Sent %s event for pod %s", eventType, pod.Name)
	default:
		logger.Warn(ctx, "Event channel full, dropping %s event for pod %s", eventType, pod.Name)
	}

	// Log scheduling events specifically
	if eventType == "pod_updated" {
		if pod.Spec.NodeName != "" && pod.Status.Phase == corev1.PodRunning {
			logger.Info(ctx, "Pod %s scheduled to node %s by Kubernetes scheduler", pod.Name, pod.Spec.NodeName)
		} else if pod.Status.Phase == corev1.PodPending {
			// Check for scheduling failures
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
					logger.Warn(ctx, "Pod %s scheduling failed: %s", pod.Name, condition.Message)
				}
			}
		}
	}
}

func (s *schedulerService) convertK8sPodToEntity(k8sPod *corev1.Pod) *entity.Pod {
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
	case corev1.PodPending:
		// Check if it's being scheduled
		if k8sPod.Spec.NodeName != "" {
			status = entity.PodStatusScheduling
		} else {
			status = entity.PodStatusPending
		}
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
