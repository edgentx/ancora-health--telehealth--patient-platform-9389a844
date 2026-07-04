// Package model holds the aggregates for the administration-and-analytics
// bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AnalyticsDashboardAggregate is the administration-and-analytics aggregate that
// tracks an analytics dashboard through its lifecycle. It embeds
// shared.AggregateRoot for version tracking and event buffering; command-handling
// behavior is added by later stories.
type AnalyticsDashboardAggregate struct {
	shared.AggregateRoot
	ID string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. This scaffold recognizes no commands yet, so every input falls
// through to shared.ErrUnknownCommand.
func (a *AnalyticsDashboardAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch cmd.(type) {
	default:
		return nil, shared.ErrUnknownCommand
	}
}
