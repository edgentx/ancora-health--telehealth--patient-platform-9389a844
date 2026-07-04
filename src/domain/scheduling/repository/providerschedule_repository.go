package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
)

// ProviderScheduleRepository is the persistence port for
// ProviderScheduleAggregate. Save persists an aggregate; FindByID loads one by
// its identity.
type ProviderScheduleRepository interface {
	Save(ctx context.Context, a *model.ProviderScheduleAggregate) error
	FindByID(ctx context.Context, id string) (*model.ProviderScheduleAggregate, error)
}
