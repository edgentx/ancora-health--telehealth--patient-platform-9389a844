// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/model"
	identityandaccessrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/repository"
)

// ErrUserAccountNotFound is returned by FindByID when no aggregate is stored
// under the requested id.
var ErrUserAccountNotFound = errors.New("user account not found")

// InMemoryUserAccountRepository is a thread-safe, map-backed implementation of
// identityandaccessrepo.UserAccountRepository for use in tests.
type InMemoryUserAccountRepository struct {
	mu       sync.RWMutex
	accounts map[string]*model.UserAccountAggregate
}

// NewInMemoryUserAccountRepository constructs an empty repository ready for use.
func NewInMemoryUserAccountRepository() *InMemoryUserAccountRepository {
	return &InMemoryUserAccountRepository{
		accounts: make(map[string]*model.UserAccountAggregate),
	}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryUserAccountRepository) Save(ctx context.Context, a *model.UserAccountAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.accounts == nil {
		r.accounts = make(map[string]*model.UserAccountAggregate)
	}
	r.accounts[a.ID] = a
	return nil
}

// FindByID returns the aggregate stored under id, or ErrUserAccountNotFound.
func (r *InMemoryUserAccountRepository) FindByID(ctx context.Context, id string) (*model.UserAccountAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.accounts[id]
	if !ok {
		return nil, ErrUserAccountNotFound
	}
	return a, nil
}

// Compile-time assertion that InMemoryUserAccountRepository satisfies the
// UserAccountRepository port.
var _ identityandaccessrepo.UserAccountRepository = (*InMemoryUserAccountRepository)(nil)
