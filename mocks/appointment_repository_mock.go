// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	schedulingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
)

// InMemoryAppointmentRepository is a thread-safe, in-memory implementation of
// schedulingrepo.AppointmentRepository for use in tests.
type InMemoryAppointmentRepository struct {
	mu    sync.RWMutex
	store map[string]*model.AppointmentAggregate
}

// NewInMemoryAppointmentRepository returns an empty in-memory repository.
func NewInMemoryAppointmentRepository() *InMemoryAppointmentRepository {
	return &InMemoryAppointmentRepository{store: make(map[string]*model.AppointmentAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryAppointmentRepository) Save(ctx context.Context, a *model.AppointmentAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.AppointmentAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the stored aggregate, or nil if none exists for id.
func (r *InMemoryAppointmentRepository) FindByID(ctx context.Context, id string) (*model.AppointmentAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that the mock satisfies the repository interface.
var _ schedulingrepo.AppointmentRepository = (*InMemoryAppointmentRepository)(nil)
