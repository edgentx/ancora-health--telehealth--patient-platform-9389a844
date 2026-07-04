package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	administrationandanalyticsrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/repository"
)

// InMemoryClinicDirectoryRepository is a concurrency-safe, map-backed
// implementation of administrationandanalyticsrepo.ClinicDirectoryRepository for
// use in tests.
type InMemoryClinicDirectoryRepository struct {
	mu    sync.RWMutex
	store map[string]*model.ClinicDirectoryAggregate
}

// NewInMemoryClinicDirectoryRepository returns an empty in-memory repository.
func NewInMemoryClinicDirectoryRepository() *InMemoryClinicDirectoryRepository {
	return &InMemoryClinicDirectoryRepository{store: make(map[string]*model.ClinicDirectoryAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryClinicDirectoryRepository) Save(_ context.Context, a *model.ClinicDirectoryAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.ClinicDirectoryAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the aggregate for id, or (nil, nil) when none is stored.
func (r *InMemoryClinicDirectoryRepository) FindByID(_ context.Context, id string) (*model.ClinicDirectoryAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that InMemoryClinicDirectoryRepository satisfies the interface.
var _ administrationandanalyticsrepo.ClinicDirectoryRepository = (*InMemoryClinicDirectoryRepository)(nil)
