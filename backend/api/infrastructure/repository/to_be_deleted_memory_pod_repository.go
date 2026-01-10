package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/repository"
)

type memoryPodRepository struct {
	pods map[string]*entity.Pod
	mu   sync.RWMutex
}

func NewMemoryPodRepository() repository.PodRepository {
	return &memoryPodRepository{
		pods: make(map[string]*entity.Pod),
	}
}

func (r *memoryPodRepository) CreatePod(ctx context.Context, pod *entity.Pod) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pods[pod.ID]; exists {
		return fmt.Errorf("pod with ID %s already exists", pod.ID)
	}

	r.pods[pod.ID] = pod
	return nil
}

func (r *memoryPodRepository) GetPod(ctx context.Context, podID string) (*entity.Pod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pod, exists := r.pods[podID]
	if !exists {
		return nil, fmt.Errorf("pod with ID %s not found", podID)
	}

	return pod, nil
}

func (r *memoryPodRepository) GetPods(ctx context.Context) ([]*entity.Pod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pods := make([]*entity.Pod, 0, len(r.pods))
	for _, pod := range r.pods {
		pods = append(pods, pod)
	}

	return pods, nil
}

func (r *memoryPodRepository) GetPodsByStatus(ctx context.Context, status entity.PodStatus) ([]*entity.Pod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pods := make([]*entity.Pod, 0)
	for _, pod := range r.pods {
		if pod.Status == status {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

func (r *memoryPodRepository) GetPodsByOwner(ctx context.Context, owner entity.PodOwner) ([]*entity.Pod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// NOTE: Owner field has been removed. Ownership is now determined by namespace.
	// This method is deprecated and should not be used.
	pods := make([]*entity.Pod, 0)
	return pods, nil
}

func (r *memoryPodRepository) UpdatePod(ctx context.Context, pod *entity.Pod) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pods[pod.ID]; !exists {
		return fmt.Errorf("pod with ID %s not found", pod.ID)
	}

	r.pods[pod.ID] = pod
	return nil
}

func (r *memoryPodRepository) DeletePod(ctx context.Context, podID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pods[podID]; !exists {
		return fmt.Errorf("pod with ID %s not found", podID)
	}

	delete(r.pods, podID)
	return nil
}
