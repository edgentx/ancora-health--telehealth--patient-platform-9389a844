package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	schedulingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
)

// ErrProviderScheduleNotFound is returned by FindByID when no aggregate is
// stored under the requested id.
var ErrProviderScheduleNotFound = errors.New("provider schedule not found")

// InMemoryProviderScheduleRepository is a thread-safe, map-backed implementation
// of schedulingrepo.ProviderScheduleRepository for use in tests.
type InMemoryProviderScheduleRepository struct {
	mu        sync.RWMutex
	schedules map[string]*model.ProviderScheduleAggregate
}

// NewInMemoryProviderScheduleRepository constructs an empty repository ready for use.
func NewInMemoryProviderScheduleRepository() *InMemoryProviderScheduleRepository {
	return &InMemoryProviderScheduleRepository{
		schedules: make(map[string]*model.ProviderScheduleAggregate),
	}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryProviderScheduleRepository) Save(ctx context.Context, a *model.ProviderScheduleAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.schedules == nil {
		r.schedules = make(map[string]*model.ProviderScheduleAggregate)
	}
	r.schedules[a.ID] = a
	return nil
}

// FindByID returns the aggregate stored under id, or ErrProviderScheduleNotFound.
func (r *InMemoryProviderScheduleRepository) FindByID(ctx context.Context, id string) (*model.ProviderScheduleAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.schedules[id]
	if !ok {
		return nil, ErrProviderScheduleNotFound
	}
	return a, nil
}

// Compile-time assertion that InMemoryProviderScheduleRepository satisfies the
// ProviderScheduleRepository port.
var _ schedulingrepo.ProviderScheduleRepository = (*InMemoryProviderScheduleRepository)(nil)
