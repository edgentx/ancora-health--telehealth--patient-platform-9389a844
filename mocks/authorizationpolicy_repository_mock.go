package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	authorizationrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/repository"
)

// InMemoryAuthorizationPolicyRepository is a concurrency-safe, map-backed
// implementation of authorizationrepo.AuthorizationPolicyRepository for use in
// tests.
type InMemoryAuthorizationPolicyRepository struct {
	mu    sync.RWMutex
	store map[string]*model.AuthorizationPolicyAggregate
}

// NewInMemoryAuthorizationPolicyRepository returns an empty in-memory repository.
func NewInMemoryAuthorizationPolicyRepository() *InMemoryAuthorizationPolicyRepository {
	return &InMemoryAuthorizationPolicyRepository{store: make(map[string]*model.AuthorizationPolicyAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryAuthorizationPolicyRepository) Save(_ context.Context, a *model.AuthorizationPolicyAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.AuthorizationPolicyAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the aggregate for id, or (nil, nil) when none is stored.
func (r *InMemoryAuthorizationPolicyRepository) FindByID(_ context.Context, id string) (*model.AuthorizationPolicyAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that InMemoryAuthorizationPolicyRepository satisfies the interface.
var _ authorizationrepo.AuthorizationPolicyRepository = (*InMemoryAuthorizationPolicyRepository)(nil)
