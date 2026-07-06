package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validClinicDirectoryAggregate() *ClinicDirectoryAggregate {
	return &ClinicDirectoryAggregate{ID: "directory-1"}
}

func validRegisterProviderCmd() RegisterProviderCmd {
	return RegisterProviderCmd{
		ProviderId:  "provider-1",
		Specialties: []string{"cardiology"},
		ClinicIds:   []string{"clinic-1"},
	}
}

func validManageSpecialtyCmd() ManageSpecialtyCmd {
	return ManageSpecialtyCmd{
		SpecialtyCode: "cardiology",
		DisplayName:   "Cardiology",
	}
}

func validConfigureClinicCmd() ConfigureClinicCmd {
	return ConfigureClinicCmd{
		ClinicIdentity: "clinic-1",
		OperatingHours: "09:00-17:00",
	}
}

// clinicDirectoryInvariantCases enumerates each shared invariant flag and the
// sentinel it yields, in the order the guards evaluate them.
func clinicDirectoryInvariantCases() []struct {
	name    string
	mutate  func(*ClinicDirectoryAggregate)
	wantErr error
} {
	return []struct {
		name    string
		mutate  func(*ClinicDirectoryAggregate)
		wantErr error
	}{
		{
			name:    "provider not bookable",
			mutate:  func(a *ClinicDirectoryAggregate) { a.ProviderNotBookable = true },
			wantErr: ErrProviderNotBookable,
		},
		{
			name:    "duplicate specialty code",
			mutate:  func(a *ClinicDirectoryAggregate) { a.DuplicateSpecialtyCode = true },
			wantErr: ErrDuplicateSpecialtyCode,
		},
		{
			name:    "clinic deactivation blocked",
			mutate:  func(a *ClinicDirectoryAggregate) { a.ClinicDeactivationBlocked = true },
			wantErr: ErrClinicDeactivationBlocked,
		},
	}
}

// assertDirectoryRejected verifies a rejected command produced the expected
// sentinel, emitted no events, buffered nothing and left the version untouched.
func assertDirectoryRejected(t *testing.T, agg *ClinicDirectoryAggregate, events []shared.DomainEvent, err, wantErr error) {
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

func TestRegisterProviderEmitsProviderRegisteredEvent(t *testing.T) {
	agg := validClinicDirectoryAggregate()
	cmd := validRegisterProviderCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(RegisterProviderCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(ProviderRegisteredEvent)
	if !ok {
		t.Fatalf("event type = %T, want ProviderRegisteredEvent", events[0])
	}
	if evt.Type() != ProviderRegisteredEventType || evt.Type() != "provider.registered" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.DirectoryID != agg.ID || evt.ProviderID != cmd.ProviderId {
		t.Fatalf("event identity not copied from command: %#v", evt)
	}
	if len(evt.Specialties) != 1 || evt.Specialties[0] != "cardiology" {
		t.Fatalf("event specialties = %#v, want [cardiology]", evt.Specialties)
	}
	if len(evt.ClinicIDs) != 1 || evt.ClinicIDs[0] != "clinic-1" {
		t.Fatalf("event clinic ids = %#v, want [clinic-1]", evt.ClinicIDs)
	}
	if len(agg.ProviderIDs) != 1 || agg.ProviderIDs[0] != "provider-1" {
		t.Fatalf("aggregate provider ids = %#v, want [provider-1]", agg.ProviderIDs)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestRegisterProviderRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     RegisterProviderCmd
		wantErr error
	}{
		{
			name:    "missing provider",
			cmd:     RegisterProviderCmd{Specialties: []string{"cardiology"}, ClinicIds: []string{"clinic-1"}},
			wantErr: ErrMissingProvider,
		},
		{
			name:    "missing specialties",
			cmd:     RegisterProviderCmd{ProviderId: "provider-1", ClinicIds: []string{"clinic-1"}},
			wantErr: ErrMissingSpecialties,
		},
		{
			name:    "missing clinics",
			cmd:     RegisterProviderCmd{ProviderId: "provider-1", Specialties: []string{"cardiology"}},
			wantErr: ErrMissingClinics,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validClinicDirectoryAggregate()
			events, err := agg.Execute(tt.cmd)
			assertDirectoryRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestRegisterProviderRejectsInvariantViolations(t *testing.T) {
	for _, tt := range clinicDirectoryInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validClinicDirectoryAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validRegisterProviderCmd())
			assertDirectoryRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestManageSpecialtyEmitsSpecialtyUpdatedEvent(t *testing.T) {
	agg := validClinicDirectoryAggregate()
	cmd := validManageSpecialtyCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(ManageSpecialtyCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(SpecialtyUpdatedEvent)
	if !ok {
		t.Fatalf("event type = %T, want SpecialtyUpdatedEvent", events[0])
	}
	if evt.Type() != SpecialtyUpdatedEventType || evt.Type() != "specialty.updated" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.DirectoryID != agg.ID || evt.SpecialtyCode != cmd.SpecialtyCode || evt.DisplayName != cmd.DisplayName {
		t.Fatalf("event payload not copied from command: %#v", evt)
	}
	if len(agg.SpecialtyCodes) != 1 || agg.SpecialtyCodes[0] != "cardiology" {
		t.Fatalf("aggregate specialty codes = %#v, want [cardiology]", agg.SpecialtyCodes)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestManageSpecialtyUpsertLeavesExistingCodeUntouched(t *testing.T) {
	agg := validClinicDirectoryAggregate()
	agg.SpecialtyCodes = []string{"cardiology"}

	events, err := agg.Execute(ManageSpecialtyCmd{SpecialtyCode: "cardiology", DisplayName: "Cardiology Updated"})
	if err != nil {
		t.Fatalf("Execute(ManageSpecialtyCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if len(agg.SpecialtyCodes) != 1 {
		t.Fatalf("expected existing specialty code registry unchanged, got %#v", agg.SpecialtyCodes)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
}

func TestManageSpecialtyRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     ManageSpecialtyCmd
		wantErr error
	}{
		{
			name:    "missing specialty code",
			cmd:     ManageSpecialtyCmd{DisplayName: "Cardiology"},
			wantErr: ErrMissingSpecialtyCode,
		},
		{
			name:    "missing display name",
			cmd:     ManageSpecialtyCmd{SpecialtyCode: "cardiology"},
			wantErr: ErrMissingDisplayName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validClinicDirectoryAggregate()
			events, err := agg.Execute(tt.cmd)
			assertDirectoryRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestManageSpecialtyRejectsInvariantViolations(t *testing.T) {
	for _, tt := range clinicDirectoryInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validClinicDirectoryAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validManageSpecialtyCmd())
			assertDirectoryRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestConfigureClinicEmitsClinicConfiguredEvent(t *testing.T) {
	agg := validClinicDirectoryAggregate()
	cmd := validConfigureClinicCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(ConfigureClinicCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(ClinicConfiguredEvent)
	if !ok {
		t.Fatalf("event type = %T, want ClinicConfiguredEvent", events[0])
	}
	if evt.Type() != ClinicConfiguredEventType || evt.Type() != "clinic.configured" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.DirectoryID != agg.ID || evt.ClinicIdentity != cmd.ClinicIdentity || evt.OperatingHours != cmd.OperatingHours {
		t.Fatalf("event payload not copied from command: %#v", evt)
	}
	if len(agg.ClinicIDs) != 1 || agg.ClinicIDs[0] != "clinic-1" {
		t.Fatalf("aggregate clinic ids = %#v, want [clinic-1]", agg.ClinicIDs)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestConfigureClinicUpsertLeavesExistingIdentityUntouched(t *testing.T) {
	agg := validClinicDirectoryAggregate()
	agg.ClinicIDs = []string{"clinic-1"}

	events, err := agg.Execute(ConfigureClinicCmd{ClinicIdentity: "clinic-1", OperatingHours: "10:00-18:00"})
	if err != nil {
		t.Fatalf("Execute(ConfigureClinicCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if len(agg.ClinicIDs) != 1 {
		t.Fatalf("expected existing clinic id registry unchanged, got %#v", agg.ClinicIDs)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
}

func TestConfigureClinicRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     ConfigureClinicCmd
		wantErr error
	}{
		{
			name:    "missing clinic identity",
			cmd:     ConfigureClinicCmd{OperatingHours: "09:00-17:00"},
			wantErr: ErrMissingClinicIdentity,
		},
		{
			name:    "missing operating hours",
			cmd:     ConfigureClinicCmd{ClinicIdentity: "clinic-1"},
			wantErr: ErrMissingOperatingHours,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validClinicDirectoryAggregate()
			events, err := agg.Execute(tt.cmd)
			assertDirectoryRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestConfigureClinicRejectsInvariantViolations(t *testing.T) {
	for _, tt := range clinicDirectoryInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validClinicDirectoryAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validConfigureClinicCmd())
			assertDirectoryRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestClinicDirectoryExecuteRejectsUnknownCommand(t *testing.T) {
	agg := validClinicDirectoryAggregate()

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

func TestClinicDirectoryAggregateRootHelpers(t *testing.T) {
	agg := validClinicDirectoryAggregate()

	if _, err := agg.Execute(validRegisterProviderCmd()); err != nil {
		t.Fatalf("Execute(RegisterProviderCmd) returned error: %v", err)
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
