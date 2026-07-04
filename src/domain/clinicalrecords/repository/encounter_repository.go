// Package repository declares persistence contracts for the clinical-records
// bounded context.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
)

// EncounterRepository persists and retrieves Encounter aggregates.
type EncounterRepository interface {
	// Save stores the aggregate, creating or updating it as needed.
	Save(ctx context.Context, a *model.EncounterAggregate) error
	// FindByID returns the aggregate with the given identity.
	FindByID(ctx context.Context, id string) (*model.EncounterAggregate, error)
}
