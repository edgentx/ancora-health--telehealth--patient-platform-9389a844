// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	patientengagementrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
)

// InMemoryPrescriptionRepository is a thread-safe, in-memory implementation of
// patientengagementrepo.PrescriptionRepository for use in tests.
type InMemoryPrescriptionRepository struct {
	mu    sync.RWMutex
	store map[string]*model.PrescriptionAggregate
}

// NewInMemoryPrescriptionRepository returns an empty in-memory repository.
func NewInMemoryPrescriptionRepository() *InMemoryPrescriptionRepository {
	return &InMemoryPrescriptionRepository{store: make(map[string]*model.PrescriptionAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryPrescriptionRepository) Save(ctx context.Context, a *model.PrescriptionAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.PrescriptionAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the aggregate for id, or (nil, nil) if none is stored.
func (r *InMemoryPrescriptionRepository) FindByID(ctx context.Context, id string) (*model.PrescriptionAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that the mock satisfies the repository interface.
var _ patientengagementrepo.PrescriptionRepository = (*InMemoryPrescriptionRepository)(nil)
