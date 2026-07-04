// Package model holds the aggregates for the authorization bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AuthorizationPolicyAggregate is the authorization aggregate that governs
// access-control policy decisions. It embeds shared.AggregateRoot for version
// tracking and event buffering, and carries its own identity in ID.
//
// This is a scaffold: it declares no commands yet, so Execute recognizes none
// and every input falls through to shared.ErrUnknownCommand. Later stories add
// command cases to the switch.
type AuthorizationPolicyAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. No command types are recognized yet, so every input returns
// shared.ErrUnknownCommand.
func (a *AuthorizationPolicyAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
