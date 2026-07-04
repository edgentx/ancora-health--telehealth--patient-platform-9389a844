// Package model holds the aggregates for the identity-and-access bounded
// context. SessionAggregate represents an authenticated user session;
// commands are dispatched through Execute.
package model

import (
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// SessionAggregate is the aggregate root for an identity-and-access session. It
// embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own identity in ID.
//
// This is a scaffold: the aggregate has no command handlers yet. Command
// stories add cases to Execute; until then every command is unrecognized.
type SessionAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. The command switch is empty in this scaffold, so any command type
// falls through to shared.ErrUnknownCommand.
func (a *SessionAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
