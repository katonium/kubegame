package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/repository"
)

type memoryNodeRepository struct {
	nodes map[string]*entity.Node
	mu    sync.RWMutex
}

func NewMemoryNodeRepository() repository.NodeRepository {
	return &memoryNodeRepository{
		nodes: make(map[string]*entity.Node),
	}
}

func (r *memoryNodeRepository) CreateNode(ctx context.Context, node *entity.Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[node.ID]; exists {
		return fmt.Errorf("node with ID %s already exists", node.ID)
	}

	r.nodes[node.ID] = node
	return nil
}

func (r *memoryNodeRepository) GetNode(ctx context.Context, nodeID string) (*entity.Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	node, exists := r.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node with ID %s not found", nodeID)
	}

	return node, nil
}

func (r *memoryNodeRepository) GetNodes(ctx context.Context) ([]*entity.Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*entity.Node, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (r *memoryNodeRepository) GetNodesByType(ctx context.Context, nodeType string) ([]*entity.Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*entity.Node, 0)
	for _, node := range r.nodes {
		if node.NodeType == nodeType {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

func (r *memoryNodeRepository) UpdateNode(ctx context.Context, node *entity.Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[node.ID]; !exists {
		return fmt.Errorf("node with ID %s not found", node.ID)
	}

	r.nodes[node.ID] = node
	return nil
}

func (r *memoryNodeRepository) DeleteNode(ctx context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[nodeID]; !exists {
		return fmt.Errorf("node with ID %s not found", nodeID)
	}

	delete(r.nodes, nodeID)
	return nil
}
