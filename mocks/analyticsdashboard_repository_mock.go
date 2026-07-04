package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	administrationandanalyticsrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/repository"
)

// InMemoryAnalyticsDashboardRepository is a concurrency-safe, map-backed
// implementation of administrationandanalyticsrepo.AnalyticsDashboardRepository
// for use in tests.
type InMemoryAnalyticsDashboardRepository struct {
	mu    sync.RWMutex
	store map[string]*model.AnalyticsDashboardAggregate
}

// NewInMemoryAnalyticsDashboardRepository returns an empty in-memory repository.
func NewInMemoryAnalyticsDashboardRepository() *InMemoryAnalyticsDashboardRepository {
	return &InMemoryAnalyticsDashboardRepository{store: make(map[string]*model.AnalyticsDashboardAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryAnalyticsDashboardRepository) Save(_ context.Context, a *model.AnalyticsDashboardAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.AnalyticsDashboardAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the aggregate for id, or (nil, nil) when none is stored.
func (r *InMemoryAnalyticsDashboardRepository) FindByID(_ context.Context, id string) (*model.AnalyticsDashboardAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that InMemoryAnalyticsDashboardRepository satisfies the interface.
var _ administrationandanalyticsrepo.AnalyticsDashboardRepository = (*InMemoryAnalyticsDashboardRepository)(nil)
