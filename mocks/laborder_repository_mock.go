// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	clinicalrecordsrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/repository"
)

// InMemoryLabOrderRepository is a concurrency-safe, map-backed implementation of
// clinicalrecordsrepo.LabOrderRepository for use in tests.
type InMemoryLabOrderRepository struct {
	mu    sync.RWMutex
	store map[string]*model.LabOrderAggregate
}

// NewInMemoryLabOrderRepository returns an empty in-memory repository.
func NewInMemoryLabOrderRepository() *InMemoryLabOrderRepository {
	return &InMemoryLabOrderRepository{store: make(map[string]*model.LabOrderAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryLabOrderRepository) Save(_ context.Context, a *model.LabOrderAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.LabOrderAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the aggregate for id, or (nil, nil) when none is stored.
func (r *InMemoryLabOrderRepository) FindByID(_ context.Context, id string) (*model.LabOrderAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that InMemoryLabOrderRepository satisfies the interface.
var _ clinicalrecordsrepo.LabOrderRepository = (*InMemoryLabOrderRepository)(nil)
