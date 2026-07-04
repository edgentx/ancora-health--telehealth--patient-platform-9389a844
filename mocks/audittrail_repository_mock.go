// Package mocks provides in-memory test doubles for the domain repositories.
package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	auditandcompliancerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/repository"
)

// ErrAuditTrailNotFound is returned by FindByID when no aggregate is stored
// under the requested id.
var ErrAuditTrailNotFound = errors.New("audit trail not found")

// InMemoryAuditTrailRepository is a thread-safe, map-backed implementation of
// auditandcompliancerepo.AuditTrailRepository for use in tests.
type InMemoryAuditTrailRepository struct {
	mu     sync.RWMutex
	trails map[string]*model.AuditTrailAggregate
}

// NewInMemoryAuditTrailRepository constructs an empty repository ready for use.
func NewInMemoryAuditTrailRepository() *InMemoryAuditTrailRepository {
	return &InMemoryAuditTrailRepository{
		trails: make(map[string]*model.AuditTrailAggregate),
	}
}

// Save stores the aggregate keyed by its ID.
func (r *InMemoryAuditTrailRepository) Save(ctx context.Context, a *model.AuditTrailAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.trails == nil {
		r.trails = make(map[string]*model.AuditTrailAggregate)
	}
	r.trails[a.ID] = a
	return nil
}

// FindByID returns the aggregate stored under id, or ErrAuditTrailNotFound.
func (r *InMemoryAuditTrailRepository) FindByID(ctx context.Context, id string) (*model.AuditTrailAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.trails[id]
	if !ok {
		return nil, ErrAuditTrailNotFound
	}
	return a, nil
}

// Compile-time assertion that InMemoryAuditTrailRepository satisfies the
// AuditTrailRepository port.
var _ auditandcompliancerepo.AuditTrailRepository = (*InMemoryAuditTrailRepository)(nil)
