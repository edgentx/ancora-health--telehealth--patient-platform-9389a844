// Package repository declares the persistence contracts for the clinical-records
// bounded context.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
)

// LabOrderRepository persists and retrieves LabOrderAggregate instances. Concrete
// implementations (database-backed or in-memory) live outside the domain layer.
type LabOrderRepository interface {
	// Save persists the aggregate, honoring optimistic-concurrency semantics.
	Save(ctx context.Context, a *model.LabOrderAggregate) error
	// FindByID loads the aggregate identified by id.
	FindByID(ctx context.Context, id string) (*model.LabOrderAggregate, error)
}
