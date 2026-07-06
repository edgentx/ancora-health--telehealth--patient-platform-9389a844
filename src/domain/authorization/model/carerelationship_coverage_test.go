package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validEstablishCareRelationshipCmd() EstablishCareRelationshipCmd {
	return EstablishCareRelationshipCmd{
		ProviderID: "provider-1",
		PatientID:  "patient-1",
		ClinicID:   "clinic-1",
	}
}

func TestEstablishCareRelationshipEmitsEstablishedEvent(t *testing.T) {
	agg := &CareRelationshipAggregate{ID: "relationship-1"}
	cmd := validEstablishCareRelationshipCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(EstablishCareRelationshipCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(CareRelationshipEstablishedEvent)
	if !ok {
		t.Fatalf("event type = %T, want CareRelationshipEstablishedEvent", events[0])
	}
	if evt.Type() != CareRelationshipEstablishedEventType {
		t.Fatalf("event type = %q, want %q", evt.Type(), CareRelationshipEstablishedEventType)
	}
	if evt.Type() != "care.relationship.established" {
		t.Fatalf("event wire name = %q, want care.relationship.established", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.RelationshipID != agg.ID {
		t.Fatalf("event relationship id = %q, want %q", evt.RelationshipID, agg.ID)
	}
	if evt.ProviderID != cmd.ProviderID || evt.PatientID != cmd.PatientID || evt.ClinicID != cmd.ClinicID {
		t.Fatalf("event payload = %#v", evt)
	}

	// Mutated state.
	if agg.Status != RelationshipStatusActive {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, RelationshipStatusActive)
	}
	if agg.ProviderID != cmd.ProviderID || agg.PatientID != cmd.PatientID || agg.ClinicID != cmd.ClinicID {
		t.Fatalf("aggregate scope = provider %q patient %q clinic %q", agg.ProviderID, agg.PatientID, agg.ClinicID)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestEstablishCareRelationshipRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     EstablishCareRelationshipCmd
		wantErr error
	}{
		{
			name:    "missing provider id",
			cmd:     EstablishCareRelationshipCmd{PatientID: "patient-1", ClinicID: "clinic-1"},
			wantErr: ErrMissingProviderID,
		},
		{
			name:    "missing patient id",
			cmd:     EstablishCareRelationshipCmd{ProviderID: "provider-1", ClinicID: "clinic-1"},
			wantErr: ErrMissingPatientID,
		},
		{
			name:    "missing clinic id",
			cmd:     EstablishCareRelationshipCmd{ProviderID: "provider-1", PatientID: "patient-1"},
			wantErr: ErrMissingClinicID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &CareRelationshipAggregate{ID: "relationship-1"}

			events, err := agg.Execute(tt.cmd)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events, got %d", len(events))
			}
			if len(agg.Events()) != 0 {
				t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
			}
			if agg.Version != 0 {
				t.Fatalf("expected version 0, got %d", agg.Version)
			}
		})
	}
}

func TestEstablishCareRelationshipRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mark    func(*CareRelationshipAggregate)
		wantErr error
	}{
		{
			name:    "inactive relationship",
			mark:    func(a *CareRelationshipAggregate) { a.Inactive = true },
			wantErr: ErrNoActiveRelationship,
		},
		{
			name:    "care episode ended",
			mark:    func(a *CareRelationshipAggregate) { a.EpisodeEnded = true },
			wantErr: ErrCareEpisodeEnded,
		},
		{
			name:    "self asserted relationship",
			mark:    func(a *CareRelationshipAggregate) { a.SelfAsserted = true },
			wantErr: ErrSelfAssertedRelationship,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &CareRelationshipAggregate{ID: "relationship-1"}
			tt.mark(agg)

			events, err := agg.Execute(validEstablishCareRelationshipCmd())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events, got %d", len(events))
			}
			if len(agg.Events()) != 0 {
				t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
			}
			if agg.Version != 0 {
				t.Fatalf("expected version 0, got %d", agg.Version)
			}
		})
	}
}

func TestRevokeCareRelationshipRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     RevokeCareRelationshipCmd
		wantErr error
	}{
		{
			name:    "missing relationship id",
			cmd:     RevokeCareRelationshipCmd{Reason: "episode ended"},
			wantErr: ErrMissingRelationshipID,
		},
		{
			name:    "missing reason",
			cmd:     RevokeCareRelationshipCmd{RelationshipID: "relationship-1"},
			wantErr: ErrMissingReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validCareRelationshipAggregate()

			events, err := agg.Execute(tt.cmd)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events, got %d", len(events))
			}
			if len(agg.Events()) != 0 {
				t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
			}
			if agg.Version != 0 {
				t.Fatalf("expected version 0, got %d", agg.Version)
			}
		})
	}
}

func TestAssignScopedRoleRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     AssignScopedRoleCmd
		wantErr error
	}{
		{
			name:    "missing account id",
			cmd:     AssignScopedRoleCmd{Role: "clinician", ClinicID: "clinic-1"},
			wantErr: ErrMissingAccountID,
		},
		{
			name:    "missing role",
			cmd:     AssignScopedRoleCmd{AccountID: "account-1", ClinicID: "clinic-1"},
			wantErr: ErrMissingRole,
		},
		{
			name:    "missing clinic id",
			cmd:     AssignScopedRoleCmd{AccountID: "account-1", Role: "clinician"},
			wantErr: ErrMissingClinicID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validCareRelationshipAggregate()

			events, err := agg.Execute(tt.cmd)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events, got %d", len(events))
			}
			if len(agg.Events()) != 0 {
				t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
			}
			if agg.Version != 0 {
				t.Fatalf("expected version 0, got %d", agg.Version)
			}
		})
	}
}

// TestRevokeCareRelationshipRejectsInactiveFlagOnActiveStatus exercises the
// second disjunct of the active-relationship guard (Status active but Inactive
// flag set), which the status-based case in the existing tests does not reach.
func TestRevokeCareRelationshipRejectsInactiveFlagOnActiveStatus(t *testing.T) {
	agg := &CareRelationshipAggregate{
		ID:       "relationship-1",
		Status:   RelationshipStatusActive,
		Inactive: true,
	}

	events, err := agg.Execute(RevokeCareRelationshipCmd{
		RelationshipID: "relationship-1",
		Reason:         "episode ended",
	})
	if !errors.Is(err, ErrNoActiveRelationship) {
		t.Fatalf("error = %v, want %v", err, ErrNoActiveRelationship)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version 0, got %d", agg.Version)
	}
}

func TestAssignScopedRoleRejectsInactiveFlagOnActiveStatus(t *testing.T) {
	agg := validCareRelationshipAggregate()
	agg.Inactive = true

	events, err := agg.Execute(validAssignScopedRoleCmd())
	if !errors.Is(err, ErrNoActiveRelationship) {
		t.Fatalf("error = %v, want %v", err, ErrNoActiveRelationship)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version 0, got %d", agg.Version)
	}
}

func TestCareRelationshipExecuteRejectsUnknownCommand(t *testing.T) {
	agg := validCareRelationshipAggregate()

	type bogusCmd struct{}

	events, err := agg.Execute(bogusCmd{})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}
	if len(agg.Events()) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version 0, got %d", agg.Version)
	}
}
