// Package model holds the aggregates for the billing-and-insurance bounded
// context. PaymentAggregate is the scaffold for a patient/payer payment;
// command handling is filled in by later stories.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PaymentAggregate is the aggregate root for a billing-and-insurance payment.
// It embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own identity in ID.
type PaymentAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. This is the canonical empty scaffold: no commands are recognized
// yet, so every input returns shared.ErrUnknownCommand.
func (a *PaymentAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
