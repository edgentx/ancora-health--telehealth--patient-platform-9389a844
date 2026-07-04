// Package model holds the Prescription aggregate for the patient-engagement
// bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PrescriptionAggregate is the aggregate root for a patient prescription. It
// embeds shared.AggregateRoot to inherit version tracking and the uncommitted
// domain-event buffer, and adds its own identity.
type PrescriptionAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate. This is a scaffold stub: no
// commands are handled yet, so every command falls through to the canonical
// empty switch and returns shared.ErrUnknownCommand.
func (a *PrescriptionAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
