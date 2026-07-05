package model

import (
	"errors"
	"testing"
)

func TestRevokeCareRelationshipEmitsRevokedEvent(t *testing.T) {
	aggregate := &CareRelationshipAggregate{}

	events, err := aggregate.Execute(RevokeCareRelationshipCmd{
		RelationshipId: "relationship-123",
		Reason:         "care episode ended",
	})
	if err != nil {
		t.Fatalf("Execute(RevokeCareRelationshipCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute(RevokeCareRelationshipCmd) emitted %d events, want 1", len(events))
	}

	event, ok := events[0].(CareRelationshipRevokedEvent)
	if !ok {
		t.Fatalf("event type = %T, want CareRelationshipRevokedEvent", events[0])
	}
	if event.Type() != CareRelationshipRevokedEventType {
		t.Fatalf("event type name = %q, want %q", event.Type(), CareRelationshipRevokedEventType)
	}
	if event.AggregateID() != "relationship-123" {
		t.Fatalf("event aggregate id = %q, want relationship-123", event.AggregateID())
	}
	if event.Reason != "care episode ended" {
		t.Fatalf("event reason = %q, want care episode ended", event.Reason)
	}
	if aggregate.Status != RelationshipStatusRevoked {
		t.Fatalf("aggregate status = %q, want %q", aggregate.Status, RelationshipStatusRevoked)
	}
	if aggregate.ID != "relationship-123" {
		t.Fatalf("aggregate id = %q, want relationship-123", aggregate.ID)
	}
	if aggregate.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", aggregate.Version)
	}
	if buffered := aggregate.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestRevokeCareRelationshipRejectsDomainInvariantViolations(t *testing.T) {
	tests := []struct {
		name      string
		aggregate CareRelationshipAggregate
		wantErr   error
	}{
		{
			name: "provider may only access PHI with active relationship",
			aggregate: CareRelationshipAggregate{
				ID:       "relationship-123",
				Inactive: true,
			},
			wantErr: ErrNoActiveRelationship,
		},
		{
			name: "care relationship must be revoked when episode ends",
			aggregate: CareRelationshipAggregate{
				ID:           "relationship-123",
				Status:       RelationshipStatusActive,
				EpisodeEnded: true,
			},
			wantErr: ErrCareEpisodeEnded,
		},
		{
			name: "relationship cannot be self asserted without governing grant",
			aggregate: CareRelationshipAggregate{
				ID:           "relationship-123",
				Status:       RelationshipStatusActive,
				SelfAsserted: true,
			},
			wantErr: ErrSelfAssertedRelationship,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aggregate := tt.aggregate

			events, err := aggregate.Execute(RevokeCareRelationshipCmd{
				RelationshipID: "relationship-123",
				Reason:         "care episode ended",
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute(RevokeCareRelationshipCmd) error = %v, want %v", err, tt.wantErr)
			}
			if len(events) != 0 {
				t.Fatalf("Execute(RevokeCareRelationshipCmd) emitted %d events, want 0", len(events))
			}
			if buffered := aggregate.Events(); len(buffered) != 0 {
				t.Fatalf("aggregate buffered %d events, want 0", len(buffered))
			}
		})
	}
}
