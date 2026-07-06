package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// scheduleInvariantCase is a shared invariant-violation scenario for the provider
// schedule aggregate: each mutation flips one invariant flag and expects the
// matching sentinel error.
type scheduleInvariantCase struct {
	name    string
	mutate  func(*ProviderScheduleAggregate)
	wantErr error
}

func scheduleInvariantCases() []scheduleInvariantCase {
	return []scheduleInvariantCase{
		{
			name:    "windows overlap",
			mutate:  func(a *ProviderScheduleAggregate) { a.WindowsOverlap = true },
			wantErr: ErrOverlappingWindows,
		},
		{
			name:    "window offers blocked interval",
			mutate:  func(a *ProviderScheduleAggregate) { a.WindowOffersBlockedInterval = true },
			wantErr: ErrBlockedIntervalOffered,
		},
		{
			name:    "window outside operating hours",
			mutate:  func(a *ProviderScheduleAggregate) { a.WindowOutsideOperatingHours = true },
			wantErr: ErrOutsideOperatingHours,
		},
	}
}

func assertScheduleRejected(t *testing.T, agg *ProviderScheduleAggregate, events []shared.DomainEvent, err error, wantErr error) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
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

func TestProviderScheduleExecutePublishAvailabilityEmitsPublishedEvent(t *testing.T) {
	agg := &ProviderScheduleAggregate{ID: "schedule-1"}
	windows := []string{"2026-07-06T09:00:00Z/2026-07-06T12:00:00Z"}

	events, err := agg.Execute(PublishAvailabilityCmd{
		ProviderId: "provider-1",
		Windows:    windows,
	})
	if err != nil {
		t.Fatalf("Execute(PublishAvailabilityCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event, ok := events[0].(ProviderAvailabilityPublishedEvent)
	if !ok {
		t.Fatalf("expected ProviderAvailabilityPublishedEvent, got %T", events[0])
	}
	if event.Type() != ProviderAvailabilityPublishedEventType || event.Type() != "provider.availability.published" {
		t.Fatalf("unexpected event type %q", event.Type())
	}
	if event.AggregateID() != "schedule-1" {
		t.Fatalf("expected aggregate id schedule-1, got %q", event.AggregateID())
	}
	if event.ProviderID != "provider-1" {
		t.Fatalf("expected provider id copied to event, got %q", event.ProviderID)
	}
	if len(event.Windows) != 1 || event.Windows[0] != windows[0] {
		t.Fatalf("expected windows copied to event, got %v", event.Windows)
	}
	if agg.ScopedProviderID != "provider-1" {
		t.Fatalf("expected schedule scoped to provider-1, got %q", agg.ScopedProviderID)
	}
	if len(agg.PublishedWindows) != 1 || agg.PublishedWindows[0] != windows[0] {
		t.Fatalf("expected published windows recorded, got %v", agg.PublishedWindows)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestProviderScheduleExecutePublishAvailabilityRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     PublishAvailabilityCmd
		wantErr error
	}{
		{
			name:    "missing provider",
			cmd:     PublishAvailabilityCmd{Windows: []string{"window"}},
			wantErr: ErrMissingScheduleProvider,
		},
		{
			name:    "missing windows nil",
			cmd:     PublishAvailabilityCmd{ProviderId: "provider-1"},
			wantErr: ErrMissingWindows,
		},
		{
			name:    "missing windows empty",
			cmd:     PublishAvailabilityCmd{ProviderId: "provider-1", Windows: []string{}},
			wantErr: ErrMissingWindows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &ProviderScheduleAggregate{ID: "schedule-1"}
			events, err := agg.Execute(tt.cmd)
			assertScheduleRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestProviderScheduleExecutePublishAvailabilityRejectsInvariantViolations(t *testing.T) {
	for _, tt := range scheduleInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &ProviderScheduleAggregate{ID: "schedule-1"}
			tt.mutate(agg)
			events, err := agg.Execute(PublishAvailabilityCmd{
				ProviderId: "provider-1",
				Windows:    []string{"window"},
			})
			assertScheduleRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestProviderScheduleExecuteBlockTimeEmitsTimeBlockedEvent(t *testing.T) {
	agg := &ProviderScheduleAggregate{ID: "schedule-1"}

	events, err := agg.Execute(BlockTimeCmd{
		ProviderId: "provider-1",
		Interval:   "2026-07-06T13:00:00Z/2026-07-06T14:00:00Z",
		Reason:     "lunch",
	})
	if err != nil {
		t.Fatalf("Execute(BlockTimeCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event, ok := events[0].(ProviderTimeBlockedEvent)
	if !ok {
		t.Fatalf("expected ProviderTimeBlockedEvent, got %T", events[0])
	}
	if event.Type() != ProviderTimeBlockedEventType || event.Type() != "provider.time.blocked" {
		t.Fatalf("unexpected event type %q", event.Type())
	}
	if event.AggregateID() != "schedule-1" {
		t.Fatalf("expected aggregate id schedule-1, got %q", event.AggregateID())
	}
	if event.ProviderID != "provider-1" || event.Interval != "2026-07-06T13:00:00Z/2026-07-06T14:00:00Z" || event.Reason != "lunch" {
		t.Fatalf("event fields not copied from command: %+v", event)
	}
	if agg.ScopedProviderID != "provider-1" {
		t.Fatalf("expected schedule scoped to provider-1, got %q", agg.ScopedProviderID)
	}
	if len(agg.BlockedIntervals) != 1 || agg.BlockedIntervals[0] != "2026-07-06T13:00:00Z/2026-07-06T14:00:00Z" {
		t.Fatalf("expected blocked interval recorded, got %v", agg.BlockedIntervals)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestProviderScheduleExecuteBlockTimeAppendsToExistingIntervals(t *testing.T) {
	agg := &ProviderScheduleAggregate{
		ID:               "schedule-1",
		BlockedIntervals: []string{"existing"},
	}

	if _, err := agg.Execute(BlockTimeCmd{
		ProviderId: "provider-1",
		Interval:   "new",
		Reason:     "leave",
	}); err != nil {
		t.Fatalf("Execute(BlockTimeCmd) returned error: %v", err)
	}

	if len(agg.BlockedIntervals) != 2 || agg.BlockedIntervals[0] != "existing" || agg.BlockedIntervals[1] != "new" {
		t.Fatalf("expected interval appended to existing, got %v", agg.BlockedIntervals)
	}
}

func TestProviderScheduleExecuteBlockTimeRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     BlockTimeCmd
		wantErr error
	}{
		{
			name:    "missing provider",
			cmd:     BlockTimeCmd{Interval: "interval", Reason: "leave"},
			wantErr: ErrMissingScheduleProvider,
		},
		{
			name:    "missing interval",
			cmd:     BlockTimeCmd{ProviderId: "provider-1", Reason: "leave"},
			wantErr: ErrMissingInterval,
		},
		{
			name:    "missing reason",
			cmd:     BlockTimeCmd{ProviderId: "provider-1", Interval: "interval"},
			wantErr: ErrMissingBlockReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &ProviderScheduleAggregate{ID: "schedule-1"}
			events, err := agg.Execute(tt.cmd)
			assertScheduleRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestProviderScheduleExecuteBlockTimeRejectsInvariantViolations(t *testing.T) {
	for _, tt := range scheduleInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &ProviderScheduleAggregate{ID: "schedule-1"}
			tt.mutate(agg)
			events, err := agg.Execute(BlockTimeCmd{
				ProviderId: "provider-1",
				Interval:   "interval",
				Reason:     "leave",
			})
			assertScheduleRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestProviderScheduleExecuteUnknownCommand(t *testing.T) {
	agg := &ProviderScheduleAggregate{ID: "schedule-1"}

	events, err := agg.Execute(struct{ Unrecognized string }{Unrecognized: "x"})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if agg.Version != 0 {
		t.Fatalf("expected version to remain 0, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(got))
	}
}

func TestProviderScheduleAggregateRootHelpers(t *testing.T) {
	agg := &ProviderScheduleAggregate{ID: "schedule-1"}

	if _, err := agg.Execute(PublishAvailabilityCmd{
		ProviderId: "provider-1",
		Windows:    []string{"window"},
	}); err != nil {
		t.Fatalf("Execute(PublishAvailabilityCmd) returned error: %v", err)
	}

	if agg.GetVersion() != 1 {
		t.Fatalf("expected GetVersion 1, got %d", agg.GetVersion())
	}
	if len(agg.Events()) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(agg.Events()))
	}

	agg.ClearEvents()
	if len(agg.Events()) != 0 {
		t.Fatalf("expected events cleared, got %d", len(agg.Events()))
	}
	if agg.GetVersion() != 1 {
		t.Fatalf("expected version unchanged after ClearEvents, got %d", agg.GetVersion())
	}
}
