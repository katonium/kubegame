package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/repository"
)

type memoryGameRepository struct {
	games map[string]*entity.Game
	pods  map[string]*entity.Pod
	nodes map[string]*entity.Node
	mu    sync.RWMutex
}

func NewMemoryGameRepository() repository.GameRepository {
	return &memoryGameRepository{
		games: make(map[string]*entity.Game),
		pods:  make(map[string]*entity.Pod),
		nodes: make(map[string]*entity.Node),
	}
}

func (r *memoryGameRepository) CreateGame(ctx context.Context, game *entity.Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.games[game.ID]; exists {
		return fmt.Errorf("game with ID %s already exists", game.ID)
	}

	r.games[game.ID] = game
	return nil
}

func (r *memoryGameRepository) GetGame(ctx context.Context, gameID string) (*entity.Game, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	game, exists := r.games[gameID]
	if !exists {
		return nil, fmt.Errorf("game with ID %s not found", gameID)
	}

	return game, nil
}

func (r *memoryGameRepository) UpdateGame(ctx context.Context, game *entity.Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.games[game.ID]; !exists {
		return fmt.Errorf("game with ID %s not found", game.ID)
	}

	r.games[game.ID] = game
	return nil
}

func (r *memoryGameRepository) DeleteGame(ctx context.Context, gameID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.games[gameID]; !exists {
		return fmt.Errorf("game with ID %s not found", gameID)
	}

	delete(r.games, gameID)
	return nil
}
