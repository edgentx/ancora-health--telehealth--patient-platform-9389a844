package model

import (
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// ProviderScheduleAggregate is the scheduling ProviderSchedule aggregate. It
// embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own string identity in ID.
//
// This is a scaffold: it recognizes no commands yet. Command handlers and the
// state they read are added by later scheduling stories.
type ProviderScheduleAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. The scaffold recognizes no commands, so every input falls through
// to shared.ErrUnknownCommand.
func (a *ProviderScheduleAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
