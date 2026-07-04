// Package repository declares the persistence contracts for the patient-engagement
// bounded context.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
)

// IntakeFormRepository persists and retrieves IntakeFormAggregate instances.
// Concrete implementations (database-backed or in-memory) live outside the
// domain layer.
type IntakeFormRepository interface {
	// Save persists the aggregate, honoring optimistic-concurrency semantics.
	Save(ctx context.Context, a *model.IntakeFormAggregate) error
	// FindByID loads the aggregate identified by id.
	FindByID(ctx context.Context, id string) (*model.IntakeFormAggregate, error)
}
