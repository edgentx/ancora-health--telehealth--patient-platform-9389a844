// Package repository declares the persistence contract for the Prescription
// aggregate in the patient-engagement bounded context.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
)

// PrescriptionRepository persists and retrieves PrescriptionAggregate roots.
type PrescriptionRepository interface {
	// Save stores the aggregate, honoring optimistic-concurrency semantics.
	Save(ctx context.Context, a *model.PrescriptionAggregate) error
	// FindByID loads the aggregate identified by id.
	FindByID(ctx context.Context, id string) (*model.PrescriptionAggregate, error)
}
