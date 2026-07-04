// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	patientengagementrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
)

// ErrMessageThreadNotFound is returned by FindByID when no aggregate is stored
// under the requested id.
var ErrMessageThreadNotFound = errors.New("message thread not found")

// InMemoryMessageThreadRepository is a thread-safe, map-backed implementation of
// patientengagementrepo.MessageThreadRepository for use in tests.
type InMemoryMessageThreadRepository struct {
	mu      sync.RWMutex
	threads map[string]*model.MessageThreadAggregate
}

// NewInMemoryMessageThreadRepository constructs an empty repository ready for use.
func NewInMemoryMessageThreadRepository() *InMemoryMessageThreadRepository {
	return &InMemoryMessageThreadRepository{
		threads: make(map[string]*model.MessageThreadAggregate),
	}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryMessageThreadRepository) Save(ctx context.Context, a *model.MessageThreadAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.threads == nil {
		r.threads = make(map[string]*model.MessageThreadAggregate)
	}
	r.threads[a.ID] = a
	return nil
}

// FindByID returns the aggregate stored under id, or ErrMessageThreadNotFound.
func (r *InMemoryMessageThreadRepository) FindByID(ctx context.Context, id string) (*model.MessageThreadAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.threads[id]
	if !ok {
		return nil, ErrMessageThreadNotFound
	}
	return a, nil
}

// Compile-time assertion that InMemoryMessageThreadRepository satisfies the
// MessageThreadRepository port.
var _ patientengagementrepo.MessageThreadRepository = (*InMemoryMessageThreadRepository)(nil)
