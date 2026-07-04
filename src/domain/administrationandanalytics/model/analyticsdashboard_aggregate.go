// Package model holds the aggregates for the administration-and-analytics
// bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AnalyticsDashboardAggregate is the administration-and-analytics aggregate that
// tracks an analytics dashboard through its lifecycle. It embeds
// shared.AggregateRoot for version tracking and event buffering, and carries its
// own identity in ID.
//
// Beyond identity it tracks the state that ComputeRollupCmd's invariants read,
// expressed as flags. Following the repository convention that a freshly
// constructed aggregate is valid, each flag's zero value is the compliant state
// and a non-zero value marks a violation the guards reject.
type AnalyticsDashboardAggregate struct {
	shared.AggregateRoot
	ID string

	// RollupOutOfScope reports that the rollup being computed would draw on events
	// outside its declared clinic/date scope. Invariant: a rollup must be computed
	// only from events within its declared clinic/date scope.
	RollupOutOfScope bool

	// ExposesPHI reports that the rollup being computed would surface
	// patient-identifiable PHI rather than an aggregate. Invariant: dashboards must
	// never expose patient-identifiable PHI, only aggregates.
	ExposesPHI bool

	// RollupNotReproducible reports that the rollup's totals cannot be reproduced
	// from its source event window. Invariant: a rollup's totals must be
	// reproducible from its source event window.
	RollupNotReproducible bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *AnalyticsDashboardAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case ComputeRollupCmd:
		return a.computeRollup(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// computeRollup handles ComputeRollupCmd: it validates the command input,
// enforces the dashboard invariants, then emits a RollupComputedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the clinic id, a complete date range, and the metric type
//     must all be present.
//   - Scope: a rollup must be computed only from events within its declared
//     clinic/date scope.
//   - PHI: dashboards must never expose patient-identifiable PHI, only
//     aggregates.
//   - Reproducibility: a rollup's totals must be reproducible from its source
//     event window.
func (a *AnalyticsDashboardAggregate) computeRollup(cmd ComputeRollupCmd) ([]shared.DomainEvent, error) {
	if cmd.ClinicId == "" {
		return nil, ErrMissingClinic
	}
	if cmd.DateRange.Start == "" || cmd.DateRange.End == "" {
		return nil, ErrMissingDateRange
	}
	if cmd.MetricType == "" {
		return nil, ErrMissingMetricType
	}

	// Invariant: a rollup must be computed only from events within its declared
	// clinic/date scope.
	if a.RollupOutOfScope {
		return nil, ErrRollupOutOfScope
	}

	// Invariant: dashboards must never expose patient-identifiable PHI, only
	// aggregates.
	if a.ExposesPHI {
		return nil, ErrPHIExposed
	}

	// Invariant: a rollup's totals must be reproducible from its source event
	// window.
	if a.RollupNotReproducible {
		return nil, ErrRollupNotReproducible
	}

	evt := RollupComputedEvent{
		DashboardID: a.ID,
		ClinicID:    cmd.ClinicId,
		RangeStart:  cmd.DateRange.Start,
		RangeEnd:    cmd.DateRange.End,
		MetricType:  cmd.MetricType,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store. A computed rollup
// records no additional state on the aggregate beyond the event itself.
func (a *AnalyticsDashboardAggregate) apply(evt RollupComputedEvent) {
	_ = evt
}
