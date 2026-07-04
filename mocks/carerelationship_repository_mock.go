package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	authorizationrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/repository"
)

// InMemoryCareRelationshipRepository is a concurrency-safe, map-backed
// implementation of authorizationrepo.CareRelationshipRepository for use in
// tests.
type InMemoryCareRelationshipRepository struct {
	mu    sync.RWMutex
	store map[string]*model.CareRelationshipAggregate
}

// NewInMemoryCareRelationshipRepository returns an empty in-memory repository.
func NewInMemoryCareRelationshipRepository() *InMemoryCareRelationshipRepository {
	return &InMemoryCareRelationshipRepository{store: make(map[string]*model.CareRelationshipAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryCareRelationshipRepository) Save(_ context.Context, a *model.CareRelationshipAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.CareRelationshipAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the aggregate for id, or (nil, nil) when none is stored.
func (r *InMemoryCareRelationshipRepository) FindByID(_ context.Context, id string) (*model.CareRelationshipAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that InMemoryCareRelationshipRepository satisfies the interface.
var _ authorizationrepo.CareRelationshipRepository = (*InMemoryCareRelationshipRepository)(nil)
