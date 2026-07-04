// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billingandinsurancerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
)

// ErrPaymentNotFound is returned by FindByID when no aggregate is stored under
// the requested id.
var ErrPaymentNotFound = errors.New("payment not found")

// InMemoryPaymentRepository is a thread-safe, map-backed implementation of
// billingandinsurancerepo.PaymentRepository for use in tests.
type InMemoryPaymentRepository struct {
	mu       sync.RWMutex
	payments map[string]*model.PaymentAggregate
}

// NewInMemoryPaymentRepository constructs an empty repository ready for use.
func NewInMemoryPaymentRepository() *InMemoryPaymentRepository {
	return &InMemoryPaymentRepository{
		payments: make(map[string]*model.PaymentAggregate),
	}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryPaymentRepository) Save(ctx context.Context, a *model.PaymentAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.payments == nil {
		r.payments = make(map[string]*model.PaymentAggregate)
	}
	r.payments[a.ID] = a
	return nil
}

// FindByID returns the aggregate stored under id, or ErrPaymentNotFound.
func (r *InMemoryPaymentRepository) FindByID(ctx context.Context, id string) (*model.PaymentAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.payments[id]
	if !ok {
		return nil, ErrPaymentNotFound
	}
	return a, nil
}

// Compile-time assertion that InMemoryPaymentRepository satisfies the
// PaymentRepository port.
var _ billingandinsurancerepo.PaymentRepository = (*InMemoryPaymentRepository)(nil)
