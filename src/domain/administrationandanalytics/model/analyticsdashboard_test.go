package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validAnalyticsDashboardAggregate() *AnalyticsDashboardAggregate {
	return &AnalyticsDashboardAggregate{ID: "dashboard-1"}
}

func validComputeRollupCmd() ComputeRollupCmd {
	return ComputeRollupCmd{
		ClinicId:   "clinic-1",
		DateRange:  DateRange{Start: "2026-01-01", End: "2026-01-31"},
		MetricType: "visit-count",
	}
}

func validQueryDashboardCmd() QueryDashboardCmd {
	return QueryDashboardCmd{
		ClinicId:   "clinic-1",
		DateRange:  DateRange{Start: "2026-01-01", End: "2026-01-31"},
		MetricType: "visit-count",
	}
}

// analyticsInvariantCases enumerates each invariant flag and the sentinel it
// yields, in the order the guards evaluate them.
func analyticsInvariantCases() []struct {
	name    string
	mutate  func(*AnalyticsDashboardAggregate)
	wantErr error
} {
	return []struct {
		name    string
		mutate  func(*AnalyticsDashboardAggregate)
		wantErr error
	}{
		{
			name:    "rollup out of scope",
			mutate:  func(a *AnalyticsDashboardAggregate) { a.RollupOutOfScope = true },
			wantErr: ErrRollupOutOfScope,
		},
		{
			name:    "exposes phi",
			mutate:  func(a *AnalyticsDashboardAggregate) { a.ExposesPHI = true },
			wantErr: ErrPHIExposed,
		},
		{
			name:    "rollup not reproducible",
			mutate:  func(a *AnalyticsDashboardAggregate) { a.RollupNotReproducible = true },
			wantErr: ErrRollupNotReproducible,
		},
	}
}

// assertDashboardRejected verifies a rejected command produced the expected
// sentinel, emitted no events, buffered nothing and left the version untouched.
func assertDashboardRejected(t *testing.T, agg *AnalyticsDashboardAggregate, events []shared.DomainEvent, err, wantErr error) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on rejection, got %d", len(events))
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events on rejection, got %d", len(got))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version to remain 0 on rejection, got %d", agg.Version)
	}
}

func TestComputeRollupEmitsRollupComputedEvent(t *testing.T) {
	agg := validAnalyticsDashboardAggregate()
	cmd := validComputeRollupCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(ComputeRollupCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(RollupComputedEvent)
	if !ok {
		t.Fatalf("event type = %T, want RollupComputedEvent", events[0])
	}
	if evt.Type() != RollupComputedEventType || evt.Type() != "analytics.rollup.computed" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.DashboardID != agg.ID {
		t.Fatalf("event dashboard id = %q, want %q", evt.DashboardID, agg.ID)
	}
	if evt.ClinicID != cmd.ClinicId || evt.RangeStart != cmd.DateRange.Start ||
		evt.RangeEnd != cmd.DateRange.End || evt.MetricType != cmd.MetricType {
		t.Fatalf("event payload not copied from command: %#v", evt)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestComputeRollupRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     ComputeRollupCmd
		wantErr error
	}{
		{
			name:    "missing clinic",
			cmd:     ComputeRollupCmd{DateRange: DateRange{Start: "2026-01-01", End: "2026-01-31"}, MetricType: "visit-count"},
			wantErr: ErrMissingClinic,
		},
		{
			name:    "missing date range start",
			cmd:     ComputeRollupCmd{ClinicId: "clinic-1", DateRange: DateRange{End: "2026-01-31"}, MetricType: "visit-count"},
			wantErr: ErrMissingDateRange,
		},
		{
			name:    "missing date range end",
			cmd:     ComputeRollupCmd{ClinicId: "clinic-1", DateRange: DateRange{Start: "2026-01-01"}, MetricType: "visit-count"},
			wantErr: ErrMissingDateRange,
		},
		{
			name:    "missing metric type",
			cmd:     ComputeRollupCmd{ClinicId: "clinic-1", DateRange: DateRange{Start: "2026-01-01", End: "2026-01-31"}},
			wantErr: ErrMissingMetricType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validAnalyticsDashboardAggregate()
			events, err := agg.Execute(tt.cmd)
			assertDashboardRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestComputeRollupRejectsInvariantViolations(t *testing.T) {
	for _, tt := range analyticsInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validAnalyticsDashboardAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validComputeRollupCmd())
			assertDashboardRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestQueryDashboardEmitsDashboardQueriedEvent(t *testing.T) {
	agg := validAnalyticsDashboardAggregate()
	cmd := validQueryDashboardCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(QueryDashboardCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(DashboardQueriedEvent)
	if !ok {
		t.Fatalf("event type = %T, want DashboardQueriedEvent", events[0])
	}
	if evt.Type() != DashboardQueriedEventType || evt.Type() != "analytics.dashboard.queried" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.DashboardID != agg.ID {
		t.Fatalf("event dashboard id = %q, want %q", evt.DashboardID, agg.ID)
	}
	if evt.ClinicID != cmd.ClinicId || evt.RangeStart != cmd.DateRange.Start ||
		evt.RangeEnd != cmd.DateRange.End || evt.MetricType != cmd.MetricType {
		t.Fatalf("event payload not copied from command: %#v", evt)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestQueryDashboardRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     QueryDashboardCmd
		wantErr error
	}{
		{
			name:    "missing clinic",
			cmd:     QueryDashboardCmd{DateRange: DateRange{Start: "2026-01-01", End: "2026-01-31"}, MetricType: "visit-count"},
			wantErr: ErrMissingClinic,
		},
		{
			name:    "missing date range start",
			cmd:     QueryDashboardCmd{ClinicId: "clinic-1", DateRange: DateRange{End: "2026-01-31"}, MetricType: "visit-count"},
			wantErr: ErrMissingDateRange,
		},
		{
			name:    "missing date range end",
			cmd:     QueryDashboardCmd{ClinicId: "clinic-1", DateRange: DateRange{Start: "2026-01-01"}, MetricType: "visit-count"},
			wantErr: ErrMissingDateRange,
		},
		{
			name:    "missing metric type",
			cmd:     QueryDashboardCmd{ClinicId: "clinic-1", DateRange: DateRange{Start: "2026-01-01", End: "2026-01-31"}},
			wantErr: ErrMissingMetricType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validAnalyticsDashboardAggregate()
			events, err := agg.Execute(tt.cmd)
			assertDashboardRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestQueryDashboardRejectsInvariantViolations(t *testing.T) {
	for _, tt := range analyticsInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validAnalyticsDashboardAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validQueryDashboardCmd())
			assertDashboardRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAnalyticsDashboardExecuteRejectsUnknownCommand(t *testing.T) {
	agg := validAnalyticsDashboardAggregate()

	type bogusCmd struct{}

	events, err := agg.Execute(bogusCmd{})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if len(agg.Events()) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version 0, got %d", agg.Version)
	}
}
