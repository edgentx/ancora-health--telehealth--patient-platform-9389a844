// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/model"
	identityandaccessrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/repository"
)

// ErrSessionNotFound is returned by FindByID when no aggregate is stored under
// the requested id.
var ErrSessionNotFound = errors.New("session not found")

// InMemorySessionRepository is a thread-safe, map-backed implementation of
// identityandaccessrepo.SessionRepository for use in tests.
type InMemorySessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]*model.SessionAggregate
}

// NewInMemorySessionRepository constructs an empty repository ready for use.
func NewInMemorySessionRepository() *InMemorySessionRepository {
	return &InMemorySessionRepository{
		sessions: make(map[string]*model.SessionAggregate),
	}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemorySessionRepository) Save(ctx context.Context, a *model.SessionAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sessions == nil {
		r.sessions = make(map[string]*model.SessionAggregate)
	}
	r.sessions[a.ID] = a
	return nil
}

// FindByID returns the aggregate stored under id, or ErrSessionNotFound.
func (r *InMemorySessionRepository) FindByID(ctx context.Context, id string) (*model.SessionAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return a, nil
}

// Compile-time assertion that InMemorySessionRepository satisfies the
// SessionRepository port.
var _ identityandaccessrepo.SessionRepository = (*InMemorySessionRepository)(nil)
