package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
)

// PrescriptionRepository is the persistence port for PrescriptionAggregate.
// Save persists an aggregate; FindByID loads one by its identity. Concrete
// adapters encrypt the prescription's PHI (medication and dosage) at rest.
type PrescriptionRepository interface {
	Save(ctx context.Context, a *model.PrescriptionAggregate) error
	FindByID(ctx context.Context, id string) (*model.PrescriptionAggregate, error)
}
