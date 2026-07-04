// Package model holds the Appointment aggregate for the scheduling bounded
// context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AppointmentAggregate is the scheduling Appointment aggregate. It embeds
// shared.AggregateRoot for version tracking and an uncommitted-event buffer,
// and carries its own string identity.
//
// This is the scaffold stub: it holds only its identity and recognizes no
// commands yet. Later scheduling stories add command handlers and the state
// their invariants read.
type AppointmentAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. The scaffold recognizes no commands yet, so every input falls
// through to shared.ErrUnknownCommand. Later stories add case arms for the
// concrete command types.
func (a *AppointmentAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
