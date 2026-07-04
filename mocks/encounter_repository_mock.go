// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	clinicalrecordsrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/repository"
)

// InMemoryEncounterRepository is a thread-safe, in-memory implementation of
// clinicalrecordsrepo.EncounterRepository for use in tests.
type InMemoryEncounterRepository struct {
	mu    sync.RWMutex
	store map[string]*model.EncounterAggregate
}

// NewInMemoryEncounterRepository returns an empty in-memory repository.
func NewInMemoryEncounterRepository() *InMemoryEncounterRepository {
	return &InMemoryEncounterRepository{store: make(map[string]*model.EncounterAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryEncounterRepository) Save(ctx context.Context, a *model.EncounterAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.EncounterAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the stored aggregate, or nil if none exists for id.
func (r *InMemoryEncounterRepository) FindByID(ctx context.Context, id string) (*model.EncounterAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that the mock satisfies the repository interface.
var _ clinicalrecordsrepo.EncounterRepository = (*InMemoryEncounterRepository)(nil)
