package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
)

// IntakeFormRepository is the persistence port for IntakeFormAggregate.
// Save persists an aggregate; FindByID loads one by its identity.
type IntakeFormRepository interface {
	Save(ctx context.Context, a *model.IntakeFormAggregate) error
	FindByID(ctx context.Context, id string) (*model.IntakeFormAggregate, error)
}
