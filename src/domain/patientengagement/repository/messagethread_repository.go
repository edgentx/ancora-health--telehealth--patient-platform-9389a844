// Package repository declares the persistence ports for the patient-engagement
// bounded context. Concrete adapters (in-memory, SQL, etc.) live elsewhere and
// implement these interfaces.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
)

// MessageThreadRepository is the persistence port for MessageThreadAggregate.
// Save persists an aggregate; FindByID loads one by its identity.
type MessageThreadRepository interface {
	Save(ctx context.Context, a *model.MessageThreadAggregate) error
	FindByID(ctx context.Context, id string) (*model.MessageThreadAggregate, error)
}
