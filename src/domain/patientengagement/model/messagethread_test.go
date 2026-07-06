package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validStartMessageThreadCmd() StartMessageThreadCmd {
	return StartMessageThreadCmd{
		PatientId:         "patient-1",
		CareTeamMemberIds: []string{"clinician-1", "clinician-2"},
		Subject:           "post-op follow up",
	}
}

func validPostSecureMessageCmd() PostSecureMessageCmd {
	return PostSecureMessageCmd{
		ThreadId: "thread-1",
		AuthorId: "clinician-1",
		Body:     "how are you feeling?",
	}
}

// messageThreadInvariantCases enumerates the shared invariant violations that
// gate every message-thread command.
func messageThreadInvariantCases() []struct {
	name    string
	mutate  func(*MessageThreadAggregate)
	wantErr error
} {
	return []struct {
		name    string
		mutate  func(*MessageThreadAggregate)
		wantErr error
	}{
		{
			name:    "access not restricted",
			mutate:  func(a *MessageThreadAggregate) { a.AccessNotRestricted = true },
			wantErr: ErrParticipantAccessNotRestricted,
		},
		{
			name:    "content not encrypted",
			mutate:  func(a *MessageThreadAggregate) { a.ContentNotEncrypted = true },
			wantErr: ErrContentNotEncrypted,
		},
		{
			name:    "no active care relationship",
			mutate:  func(a *MessageThreadAggregate) { a.NoActiveCareRelationship = true },
			wantErr: ErrNoActiveCareRelationship,
		},
	}
}

// assertThreadRejected checks that a command execution produced the expected
// sentinel error, emitted no events, buffered nothing and left the version
// untouched.
func assertThreadRejected(t *testing.T, agg *MessageThreadAggregate, events []shared.DomainEvent, err error, wantErr error) {
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

func TestMessageThreadExecuteStartEmitsStartedEvent(t *testing.T) {
	agg := &MessageThreadAggregate{ID: "thread-1"}
	cmd := validStartMessageThreadCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(StartMessageThreadCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(MessageThreadStartedEvent)
	if !ok {
		t.Fatalf("expected MessageThreadStartedEvent, got %T", events[0])
	}
	if evt.Type() != MessageThreadStartedEventType || evt.Type() != "message.thread.started" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "thread-1" {
		t.Fatalf("expected aggregate id thread-1, got %q", evt.AggregateID())
	}
	if evt.ThreadID != "thread-1" || evt.PatientID != cmd.PatientId || evt.Subject != cmd.Subject {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}
	if len(evt.CareTeamMemberIDs) != 2 || evt.CareTeamMemberIDs[0] != "clinician-1" || evt.CareTeamMemberIDs[1] != "clinician-2" {
		t.Fatalf("event care team not copied: %+v", evt.CareTeamMemberIDs)
	}

	if agg.Status != MessageThreadStatusOpen {
		t.Fatalf("expected status %q, got %q", MessageThreadStatusOpen, agg.Status)
	}
	if agg.ScopedPatientID != cmd.PatientId || agg.Subject != cmd.Subject {
		t.Fatalf("aggregate not scoped to thread: %+v", agg)
	}
	if len(agg.ScopedCareTeamMemberIDs) != 2 {
		t.Fatalf("aggregate care team not scoped: %+v", agg.ScopedCareTeamMemberIDs)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestMessageThreadExecuteStartRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     StartMessageThreadCmd
		wantErr error
	}{
		{
			name:    "missing patient",
			cmd:     StartMessageThreadCmd{CareTeamMemberIds: []string{"clinician-1"}, Subject: "s"},
			wantErr: ErrMissingThreadPatient,
		},
		{
			name:    "missing care team",
			cmd:     StartMessageThreadCmd{PatientId: "patient-1", Subject: "s"},
			wantErr: ErrMissingCareTeam,
		},
		{
			name:    "missing subject",
			cmd:     StartMessageThreadCmd{PatientId: "patient-1", CareTeamMemberIds: []string{"clinician-1"}},
			wantErr: ErrMissingSubject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &MessageThreadAggregate{ID: "thread-1"}
			events, err := agg.Execute(tt.cmd)
			assertThreadRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestMessageThreadExecuteStartRejectsInvariantViolations(t *testing.T) {
	for _, tt := range messageThreadInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &MessageThreadAggregate{ID: "thread-1"}
			tt.mutate(agg)
			events, err := agg.Execute(validStartMessageThreadCmd())
			assertThreadRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestMessageThreadExecutePostEmitsPostedEvent(t *testing.T) {
	agg := &MessageThreadAggregate{ID: "thread-1", Status: MessageThreadStatusOpen}
	cmd := validPostSecureMessageCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(PostSecureMessageCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(MessageSecurePostedEvent)
	if !ok {
		t.Fatalf("expected MessageSecurePostedEvent, got %T", events[0])
	}
	if evt.Type() != MessageSecurePostedEventType || evt.Type() != "message.secure.posted" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "thread-1" {
		t.Fatalf("expected aggregate id thread-1, got %q", evt.AggregateID())
	}
	if evt.ThreadID != "thread-1" || evt.AuthorID != cmd.AuthorId || evt.Body != cmd.Body {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}

	if agg.PostedMessageCount != 1 {
		t.Fatalf("expected posted message count 1, got %d", agg.PostedMessageCount)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestMessageThreadExecutePostRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     PostSecureMessageCmd
		wantErr error
	}{
		{
			name:    "missing thread",
			cmd:     PostSecureMessageCmd{AuthorId: "clinician-1", Body: "b"},
			wantErr: ErrMissingThread,
		},
		{
			name:    "missing author",
			cmd:     PostSecureMessageCmd{ThreadId: "thread-1", Body: "b"},
			wantErr: ErrMissingAuthor,
		},
		{
			name:    "missing body",
			cmd:     PostSecureMessageCmd{ThreadId: "thread-1", AuthorId: "clinician-1"},
			wantErr: ErrMissingBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &MessageThreadAggregate{ID: "thread-1", Status: MessageThreadStatusOpen}
			events, err := agg.Execute(tt.cmd)
			assertThreadRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestMessageThreadExecutePostRejectsInvariantViolations(t *testing.T) {
	for _, tt := range messageThreadInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &MessageThreadAggregate{ID: "thread-1", Status: MessageThreadStatusOpen}
			tt.mutate(agg)
			events, err := agg.Execute(validPostSecureMessageCmd())
			assertThreadRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestMessageThreadExecuteUnknownCommand(t *testing.T) {
	agg := &MessageThreadAggregate{ID: "thread-1"}

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
