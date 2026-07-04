// Package model holds the aggregates for the identity-and-access bounded
// context. UserAccountAggregate is the scaffold for a platform user account;
// command handling is filled in by later stories.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// UserAccountAggregate is the aggregate root for an identity-and-access user
// account. It embeds shared.AggregateRoot for version tracking and an
// uncommitted-event buffer, and carries its own identity in ID.
type UserAccountAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. This is the canonical empty scaffold: no commands are recognized
// yet, so every input returns shared.ErrUnknownCommand.
func (a *UserAccountAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
