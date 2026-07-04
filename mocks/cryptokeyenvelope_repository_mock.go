// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	auditandcompliancerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/repository"
)

// ErrCryptoKeyEnvelopeNotFound is returned by FindByID when no aggregate is
// stored under the requested id.
var ErrCryptoKeyEnvelopeNotFound = errors.New("crypto key envelope not found")

// InMemoryCryptoKeyEnvelopeRepository is a thread-safe, map-backed implementation
// of auditandcompliancerepo.CryptoKeyEnvelopeRepository for use in tests.
type InMemoryCryptoKeyEnvelopeRepository struct {
	mu        sync.RWMutex
	envelopes map[string]*model.CryptoKeyEnvelopeAggregate
}

// NewInMemoryCryptoKeyEnvelopeRepository constructs an empty repository ready for use.
func NewInMemoryCryptoKeyEnvelopeRepository() *InMemoryCryptoKeyEnvelopeRepository {
	return &InMemoryCryptoKeyEnvelopeRepository{
		envelopes: make(map[string]*model.CryptoKeyEnvelopeAggregate),
	}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryCryptoKeyEnvelopeRepository) Save(ctx context.Context, a *model.CryptoKeyEnvelopeAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.envelopes == nil {
		r.envelopes = make(map[string]*model.CryptoKeyEnvelopeAggregate)
	}
	r.envelopes[a.ID] = a
	return nil
}

// FindByID returns the aggregate stored under id, or ErrCryptoKeyEnvelopeNotFound.
func (r *InMemoryCryptoKeyEnvelopeRepository) FindByID(ctx context.Context, id string) (*model.CryptoKeyEnvelopeAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.envelopes[id]
	if !ok {
		return nil, ErrCryptoKeyEnvelopeNotFound
	}
	return a, nil
}

// Compile-time assertion that InMemoryCryptoKeyEnvelopeRepository satisfies the
// CryptoKeyEnvelopeRepository port.
var _ auditandcompliancerepo.CryptoKeyEnvelopeRepository = (*InMemoryCryptoKeyEnvelopeRepository)(nil)
