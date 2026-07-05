package model

import (
	"errors"
	"testing"
)

func TestAssignScopedRoleEmitsRoleAssignedEvent(t *testing.T) {
	agg := validCareRelationshipAggregate()
	cmd := validAssignScopedRoleCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(RoleAssignedEvent)
	if !ok {
		t.Fatalf("expected RoleAssignedEvent, got %T", events[0])
	}
	if evt.Type() != RoleAssignedEventType {
		t.Fatalf("expected event type %q, got %q", RoleAssignedEventType, evt.Type())
	}
	if evt.Type() != "authz.role.assigned" {
		t.Fatalf("expected authz.role.assigned event, got %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("expected aggregate id %q, got %q", agg.ID, evt.AggregateID())
	}
	if evt.AccountID != cmd.AccountID || evt.Role != cmd.Role || evt.ClinicID != cmd.ClinicID {
		t.Fatalf("event payload = %#v, want account %q role %q clinic %q", evt, cmd.AccountID, cmd.Role, cmd.ClinicID)
	}
	if agg.ScopedRoleAccountID != cmd.AccountID || agg.ScopedRole != cmd.Role || agg.ScopedRoleClinicID != cmd.ClinicID {
		t.Fatalf("aggregate role scope = account %q role %q clinic %q", agg.ScopedRoleAccountID, agg.ScopedRole, agg.ScopedRoleClinicID)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if len(agg.Events()) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(agg.Events()))
	}
}

func TestAssignScopedRoleRejectsCareRelationshipInvariantViolations(t *testing.T) {
	tests := []struct {
		name string
		mark func(*CareRelationshipAggregate)
		want error
	}{
		{
			name: "no active care relationship",
			mark: func(agg *CareRelationshipAggregate) {
				agg.Inactive = true
			},
			want: ErrNoActiveRelationship,
		},
		{
			name: "care episode ended",
			mark: func(agg *CareRelationshipAggregate) {
				agg.EpisodeEnded = true
			},
			want: ErrCareEpisodeEnded,
		},
		{
			name: "self asserted relationship",
			mark: func(agg *CareRelationshipAggregate) {
				agg.SelfAsserted = true
			},
			want: ErrSelfAssertedRelationship,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validCareRelationshipAggregate()
			tt.mark(agg)

			events, err := agg.Execute(validAssignScopedRoleCmd())
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected error %v, got %v", tt.want, err)
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

func validCareRelationshipAggregate() *CareRelationshipAggregate {
	return &CareRelationshipAggregate{
		ID:         "care-relationship-1",
		Status:     RelationshipStatusActive,
		ProviderID: "provider-1",
		PatientID:  "patient-1",
		ClinicID:   "clinic-1",
	}
}

func validAssignScopedRoleCmd() AssignScopedRoleCmd {
	return AssignScopedRoleCmd{
		AccountID: "account-1",
		Role:      "clinician",
		ClinicID:  "clinic-1",
	}
}
