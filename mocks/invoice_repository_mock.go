package mocks

import (
	"context"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billingandinsurancerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
)

// InMemoryInvoiceRepository is a concurrency-safe, map-backed implementation of
// billingandinsurancerepo.InvoiceRepository for use in tests.
type InMemoryInvoiceRepository struct {
	mu    sync.RWMutex
	store map[string]*model.InvoiceAggregate
}

// NewInMemoryInvoiceRepository returns an empty in-memory repository.
func NewInMemoryInvoiceRepository() *InMemoryInvoiceRepository {
	return &InMemoryInvoiceRepository{store: make(map[string]*model.InvoiceAggregate)}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryInvoiceRepository) Save(_ context.Context, a *model.InvoiceAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		r.store = make(map[string]*model.InvoiceAggregate)
	}
	r.store[a.ID] = a
	return nil
}

// FindByID returns the aggregate for id, or (nil, nil) when none is stored.
func (r *InMemoryInvoiceRepository) FindByID(_ context.Context, id string) (*model.InvoiceAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store[id], nil
}

// Compile-time assertion that InMemoryInvoiceRepository satisfies the interface.
var _ billingandinsurancerepo.InvoiceRepository = (*InMemoryInvoiceRepository)(nil)
