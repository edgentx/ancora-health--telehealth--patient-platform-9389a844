// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	patientengagementrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
)

// InMemoryIntakeFormRepository is a concurrency-safe, map-backed implementation
// of patientengagementrepo.IntakeFormRepository for use in tests.
type InMemoryIntakeFormRepository struct {
	mu    sync.RWMutex
	store map[string]*model.IntakeFormAggregate
}

// NewInMemoryIntakeFormRepository returns an empty in-memory repository.
func NewInMemoryIntakeFormRepository() *InMemoryIntakeFormRepository {
	return &InMemoryIntakeFormRepository{store: make(map[string]*model.IntakeFormAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryIntakeFormRepository) Save(_ context.Context, a *model.IntakeFormAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.IntakeFormAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the aggregate for id, or (nil, nil) when none is stored.
func (r *InMemoryIntakeFormRepository) FindByID(_ context.Context, id string) (*model.IntakeFormAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that InMemoryIntakeFormRepository satisfies the interface.
var _ patientengagementrepo.IntakeFormRepository = (*InMemoryIntakeFormRepository)(nil)
