package repository

import (
	"context"
	"github.com/katonium/kubegame/backend/domain/entity"
)

type GameRepository interface {
	CreateGame(ctx context.Context, game *entity.Game) error
	GetGame(ctx context.Context, gameID string) (*entity.Game, error)
	UpdateGame(ctx context.Context, game *entity.Game) error
	DeleteGame(ctx context.Context, gameID string) error
}

type PodRepository interface {
	CreatePod(ctx context.Context, pod *entity.Pod) error
	GetPod(ctx context.Context, podID string) (*entity.Pod, error)
	GetPods(ctx context.Context) ([]*entity.Pod, error)
	GetPodsByStatus(ctx context.Context, status entity.PodStatus) ([]*entity.Pod, error)
	GetPodsByOwner(ctx context.Context, owner entity.PodOwner) ([]*entity.Pod, error)
	UpdatePod(ctx context.Context, pod *entity.Pod) error
	DeletePod(ctx context.Context, podID string) error
}

type NodeRepository interface {
	CreateNode(ctx context.Context, node *entity.Node) error
	GetNode(ctx context.Context, nodeID string) (*entity.Node, error)
	GetNodes(ctx context.Context) ([]*entity.Node, error)
	GetNodesByType(ctx context.Context, nodeType string) ([]*entity.Node, error)
	UpdateNode(ctx context.Context, node *entity.Node) error
	DeleteNode(ctx context.Context, nodeID string) error
}
