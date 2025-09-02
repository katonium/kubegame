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
	mu    sync.RWMutex
}

func NewMemoryGameRepository() repository.GameRepository {
	return &memoryGameRepository{
		games: make(map[string]*entity.Game),
	}
}

func (r *memoryGameRepository) Put(ctx context.Context, game *entity.Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.games[game.ID]; exists {
		return fmt.Errorf("game with ID %s already exists", game.ID)
	}

	r.games[game.ID] = game
	return nil
}

func (r *memoryGameRepository) Get(ctx context.Context, gameID string) (*entity.Game, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	game, exists := r.games[gameID]
	if !exists {
		return nil, fmt.Errorf("game with ID %s not found", gameID)
	}

	return game, nil
}

func (r *memoryGameRepository) Update(ctx context.Context, game *entity.Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.games[game.ID]; !exists {
		return fmt.Errorf("game with ID %s not found", game.ID)
	}

	r.games[game.ID] = game
	return nil
}

func (r *memoryGameRepository) Delete(ctx context.Context, gameID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.games[gameID]; !exists {
		return fmt.Errorf("game with ID %s not found", gameID)
	}

	delete(r.games, gameID)
	return nil
}
