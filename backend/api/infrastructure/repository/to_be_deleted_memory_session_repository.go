package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/repository"
)

type memorySessionRepository struct {
	sessions     map[string]*entity.GameSession // indexed by connectionID
	sessionsByID map[string]*entity.GameSession // indexed by sessionID
	mu           sync.RWMutex
}

func NewMemorySessionRepository() repository.GameSessionRepository {
	return &memorySessionRepository{
		sessions:     make(map[string]*entity.GameSession),
		sessionsByID: make(map[string]*entity.GameSession),
	}
}

func (r *memorySessionRepository) CreateSession(ctx context.Context, session *entity.GameSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[session.ConnectionID]; exists {
		return fmt.Errorf("session with connection ID %s already exists", session.ConnectionID)
	}

	if _, exists := r.sessionsByID[session.SessionID]; exists {
		return fmt.Errorf("session with session ID %s already exists", session.SessionID)
	}

	r.sessions[session.ConnectionID] = session
	r.sessionsByID[session.SessionID] = session
	return nil
}

func (r *memorySessionRepository) GetSession(ctx context.Context, connectionID string) (*entity.GameSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[connectionID]
	if !exists {
		return nil, fmt.Errorf("session with connection ID %s not found", connectionID)
	}

	return session, nil
}

func (r *memorySessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*entity.GameSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessionsByID[sessionID]
	if !exists {
		return nil, fmt.Errorf("session with session ID %s not found", sessionID)
	}

	return session, nil
}

func (r *memorySessionRepository) GetAllSessions(ctx context.Context) ([]*entity.GameSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*entity.GameSession, 0, len(r.sessions))
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (r *memorySessionRepository) UpdateSession(ctx context.Context, session *entity.GameSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[session.ConnectionID]; !exists {
		return fmt.Errorf("session with connection ID %s not found", session.ConnectionID)
	}

	r.sessions[session.ConnectionID] = session
	r.sessionsByID[session.SessionID] = session
	return nil
}

func (r *memorySessionRepository) DeleteSession(ctx context.Context, connectionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[connectionID]
	if !exists {
		return fmt.Errorf("session with connection ID %s not found", connectionID)
	}

	delete(r.sessions, connectionID)
	delete(r.sessionsByID, session.SessionID)
	return nil
}

func (r *memorySessionRepository) CleanupInactiveSessions(ctx context.Context, timeout time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-timeout)
	var toDelete []string

	for connectionID, session := range r.sessions {
		if session.LastActivity.Before(cutoff) {
			toDelete = append(toDelete, connectionID)
		}
	}

	for _, connectionID := range toDelete {
		session := r.sessions[connectionID]
		delete(r.sessions, connectionID)
		delete(r.sessionsByID, session.SessionID)
	}

	return nil
}
