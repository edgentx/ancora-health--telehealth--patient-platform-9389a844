// Package model holds the aggregates for the clinical-records bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// LabOrderAggregate is the clinical-records aggregate that tracks a lab order
// through its lifecycle. It embeds shared.AggregateRoot for version tracking and
// event buffering; command-handling behavior is added by later stories.
type LabOrderAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. This scaffold recognizes no commands yet, so every input falls
// through to shared.ErrUnknownCommand.
func (a *LabOrderAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
