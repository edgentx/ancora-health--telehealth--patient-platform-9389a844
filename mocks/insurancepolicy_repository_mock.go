// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billingandinsurancerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
)

// ErrInsurancePolicyNotFound is returned by FindByID when no aggregate is stored
// under the requested id.
var ErrInsurancePolicyNotFound = errors.New("insurance policy not found")

// InMemoryInsurancePolicyRepository is a thread-safe, map-backed implementation of
// billingandinsurancerepo.InsurancePolicyRepository for use in tests.
type InMemoryInsurancePolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]*model.InsurancePolicyAggregate
}

// NewInMemoryInsurancePolicyRepository constructs an empty repository ready for use.
func NewInMemoryInsurancePolicyRepository() *InMemoryInsurancePolicyRepository {
	return &InMemoryInsurancePolicyRepository{
		policies: make(map[string]*model.InsurancePolicyAggregate),
	}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryInsurancePolicyRepository) Save(ctx context.Context, a *model.InsurancePolicyAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.policies == nil {
		r.policies = make(map[string]*model.InsurancePolicyAggregate)
	}
	r.policies[a.ID] = a
	return nil
}

// FindByID returns the aggregate stored under id, or ErrInsurancePolicyNotFound.
func (r *InMemoryInsurancePolicyRepository) FindByID(ctx context.Context, id string) (*model.InsurancePolicyAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.policies[id]
	if !ok {
		return nil, ErrInsurancePolicyNotFound
	}
	return a, nil
}

// Compile-time assertion that InMemoryInsurancePolicyRepository satisfies the
// InsurancePolicyRepository port.
var _ billingandinsurancerepo.InsurancePolicyRepository = (*InMemoryInsurancePolicyRepository)(nil)
