// Package repository declares persistence contracts for the scheduling bounded
// context.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
)

// AppointmentRepository persists and retrieves Appointment aggregates.
type AppointmentRepository interface {
	// Save stores the aggregate, creating or updating it as needed.
	Save(ctx context.Context, a *model.AppointmentAggregate) error
	// FindByID returns the aggregate with the given identity.
	FindByID(ctx context.Context, id string) (*model.AppointmentAggregate, error)
}
