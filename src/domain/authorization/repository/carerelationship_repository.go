// Package repository declares the persistence contracts for the authorization
// bounded context.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
)

// CareRelationshipRepository persists and retrieves CareRelationshipAggregate
// instances. Concrete implementations (database-backed or in-memory) live
// outside the domain layer.
type CareRelationshipRepository interface {
	// Save persists the aggregate, honoring optimistic-concurrency semantics.
	Save(ctx context.Context, a *model.CareRelationshipAggregate) error
	// FindByID loads the aggregate identified by id.
	FindByID(ctx context.Context, id string) (*model.CareRelationshipAggregate, error)
}
