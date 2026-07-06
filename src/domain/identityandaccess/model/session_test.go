package model

import (
	"errors"
	"testing"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func TestIssueSessionEmitsIssuedEvent(t *testing.T) {
	aggregate := &SessionAggregate{ID: "session-1", Authenticated: true}
	cmd := IssueSessionCmd{
		AccountId:         "account-7",
		Role:              "clinician",
		DeviceFingerprint: "device-abc",
		RequestedLifetime: 2 * time.Hour,
	}

	before := time.Now()
	events, err := aggregate.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}

	event, ok := events[0].(SessionIssuedEvent)
	if !ok {
		t.Fatalf("event type = %T, want SessionIssuedEvent", events[0])
	}
	if event.Type() != "session.issued" {
		t.Fatalf("event Type() = %q", event.Type())
	}
	if event.AggregateID() != aggregate.ID {
		t.Fatalf("event AggregateID() = %q, want %q", event.AggregateID(), aggregate.ID)
	}
	if event.SessionID != aggregate.ID {
		t.Fatalf("event SessionID = %q, want %q", event.SessionID, aggregate.ID)
	}
	if event.AccountID != cmd.AccountId {
		t.Fatalf("event AccountID = %q, want %q", event.AccountID, cmd.AccountId)
	}
	if event.Role != cmd.Role {
		t.Fatalf("event Role = %q, want %q", event.Role, cmd.Role)
	}
	if event.DeviceFingerprint != cmd.DeviceFingerprint {
		t.Fatalf("event DeviceFingerprint = %q, want %q", event.DeviceFingerprint, cmd.DeviceFingerprint)
	}
	if event.IssuedAt.Before(before) {
		t.Fatalf("event IssuedAt = %v, want >= %v", event.IssuedAt, before)
	}
	if !event.ExpiresAt.Equal(event.IssuedAt.Add(cmd.RequestedLifetime)) {
		t.Fatalf("event ExpiresAt = %v, want IssuedAt + %v", event.ExpiresAt, cmd.RequestedLifetime)
	}

	// Mutated aggregate state (via apply).
	if !aggregate.Issued {
		t.Fatal("aggregate Issued = false, want true")
	}
	if aggregate.AccountID != cmd.AccountId {
		t.Fatalf("aggregate AccountID = %q, want %q", aggregate.AccountID, cmd.AccountId)
	}
	if aggregate.Role != cmd.Role {
		t.Fatalf("aggregate Role = %q, want %q", aggregate.Role, cmd.Role)
	}
	if aggregate.DeviceFingerprint != cmd.DeviceFingerprint {
		t.Fatalf("aggregate DeviceFingerprint = %q, want %q", aggregate.DeviceFingerprint, cmd.DeviceFingerprint)
	}
	if !aggregate.ExpiresAt.Equal(event.ExpiresAt) {
		t.Fatalf("aggregate ExpiresAt = %v, want %v", aggregate.ExpiresAt, event.ExpiresAt)
	}
	if aggregate.Version != 1 {
		t.Fatalf("aggregate Version = %d, want 1", aggregate.Version)
	}
	if len(aggregate.Events()) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(aggregate.Events()))
	}
}

func TestIssueSessionRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		cmd     IssueSessionCmd
		wantErr error
	}{
		{
			name:    "missing account id",
			cmd:     IssueSessionCmd{AccountId: "  ", Role: "patient", DeviceFingerprint: "d", RequestedLifetime: time.Hour},
			wantErr: ErrMissingSessionAccountID,
		},
		{
			name:    "missing role",
			cmd:     IssueSessionCmd{AccountId: "a", Role: "", DeviceFingerprint: "d", RequestedLifetime: time.Hour},
			wantErr: ErrMissingSessionRole,
		},
		{
			name:    "missing device fingerprint",
			cmd:     IssueSessionCmd{AccountId: "a", Role: "patient", DeviceFingerprint: " ", RequestedLifetime: time.Hour},
			wantErr: ErrMissingDeviceFingerprint,
		},
		{
			name:    "non-positive lifetime",
			cmd:     IssueSessionCmd{AccountId: "a", Role: "patient", DeviceFingerprint: "d", RequestedLifetime: 0},
			wantErr: ErrMissingSessionLifetime,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := &SessionAggregate{ID: "session-1", Authenticated: true}

			events, err := aggregate.Execute(test.cmd)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoSessionMutation(t, aggregate, events)
		})
	}
}

func TestIssueSessionRejectsInvariantViolations(t *testing.T) {
	base := IssueSessionCmd{AccountId: "a", Role: "patient", DeviceFingerprint: "d", RequestedLifetime: time.Hour}

	tests := []struct {
		name      string
		aggregate SessionAggregate
		cmd       IssueSessionCmd
		wantErr   error
	}{
		{
			name:      "account not authenticated",
			aggregate: SessionAggregate{ID: "session-1", Authenticated: false},
			cmd:       base,
			wantErr:   ErrAccountNotAuthenticated,
		},
		{
			name:      "session revoked",
			aggregate: SessionAggregate{ID: "session-1", Authenticated: true, Revoked: true},
			cmd:       base,
			wantErr:   ErrSessionRevoked,
		},
		{
			name:      "lifetime exceeds per-role maximum",
			aggregate: SessionAggregate{ID: "session-1", Authenticated: true},
			cmd:       IssueSessionCmd{AccountId: "a", Role: "admin", DeviceFingerprint: "d", RequestedLifetime: 2 * time.Hour},
			wantErr:   ErrExpiryExceedsMaxLifetime,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := test.aggregate

			events, err := aggregate.Execute(test.cmd)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoSessionMutation(t, &aggregate, events)
		})
	}
}

func TestRevokeSessionEmitsRevokedEvent(t *testing.T) {
	aggregate := &SessionAggregate{ID: "session-1", Authenticated: true, Role: "patient"}
	cmd := RevokeSessionCmd{SessionId: "session-1", Reason: "logout"}

	before := time.Now()
	events, err := aggregate.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}

	event, ok := events[0].(SessionRevokedEvent)
	if !ok {
		t.Fatalf("event type = %T, want SessionRevokedEvent", events[0])
	}
	if event.Type() != "session.revoked" {
		t.Fatalf("event Type() = %q", event.Type())
	}
	if event.AggregateID() != aggregate.ID {
		t.Fatalf("event AggregateID() = %q, want %q", event.AggregateID(), aggregate.ID)
	}
	if event.SessionID != aggregate.ID {
		t.Fatalf("event SessionID = %q, want %q", event.SessionID, aggregate.ID)
	}
	if event.Reason != cmd.Reason {
		t.Fatalf("event Reason = %q, want %q", event.Reason, cmd.Reason)
	}
	if event.RevokedAt.Before(before) {
		t.Fatalf("event RevokedAt = %v, want >= %v", event.RevokedAt, before)
	}
	if !aggregate.Revoked {
		t.Fatal("aggregate Revoked = false, want true")
	}
	if aggregate.Version != 1 {
		t.Fatalf("aggregate Version = %d, want 1", aggregate.Version)
	}
	if len(aggregate.Events()) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(aggregate.Events()))
	}
}

func TestRevokeSessionEmitsRevokedEventWithinRoleLifetime(t *testing.T) {
	// A session whose remaining life is within the per-role ceiling passes the
	// expiry invariant and is revoked successfully.
	aggregate := &SessionAggregate{
		ID:            "session-1",
		Authenticated: true,
		Role:          "admin",
		ExpiresAt:     time.Now().Add(30 * time.Minute),
	}

	events, err := aggregate.Execute(RevokeSessionCmd{SessionId: "session-1", Reason: "logout"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}
	if !aggregate.Revoked {
		t.Fatal("aggregate Revoked = false, want true")
	}
}

func TestRevokeSessionRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		cmd     RevokeSessionCmd
		wantErr error
	}{
		{
			name:    "missing session id",
			cmd:     RevokeSessionCmd{SessionId: "  ", Reason: "logout"},
			wantErr: ErrMissingSessionID,
		},
		{
			name:    "missing revocation reason",
			cmd:     RevokeSessionCmd{SessionId: "session-1", Reason: ""},
			wantErr: ErrMissingRevocationReason,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := &SessionAggregate{ID: "session-1", Authenticated: true}

			events, err := aggregate.Execute(test.cmd)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoSessionMutation(t, aggregate, events)
		})
	}
}

func TestRevokeSessionRejectsInvariantViolations(t *testing.T) {
	cmd := RevokeSessionCmd{SessionId: "session-1", Reason: "logout"}

	tests := []struct {
		name      string
		aggregate SessionAggregate
		wantErr   error
	}{
		{
			name:      "account not authenticated",
			aggregate: SessionAggregate{ID: "session-1", Authenticated: false},
			wantErr:   ErrAccountNotAuthenticated,
		},
		{
			name:      "session already revoked",
			aggregate: SessionAggregate{ID: "session-1", Authenticated: true, Revoked: true},
			wantErr:   ErrSessionRevoked,
		},
		{
			name: "remaining life exceeds per-role maximum",
			aggregate: SessionAggregate{
				ID:            "session-1",
				Authenticated: true,
				Role:          "admin",
				ExpiresAt:     time.Now().Add(2 * time.Hour),
			},
			wantErr: ErrExpiryExceedsMaxLifetime,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := test.aggregate

			events, err := aggregate.Execute(cmd)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoSessionMutation(t, &aggregate, events)
		})
	}
}

func TestSessionExecuteRejectsUnknownCommand(t *testing.T) {
	aggregate := &SessionAggregate{ID: "session-1", Authenticated: true}

	type bogusCmd struct{}
	events, err := aggregate.Execute(bogusCmd{})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("Execute() error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	assertNoSessionMutation(t, aggregate, events)
}

func TestMaxLifetimeForRole(t *testing.T) {
	tests := []struct {
		role string
		want time.Duration
	}{
		{"admin", 1 * time.Hour},
		{"administrator", 1 * time.Hour},
		{"clinician", 8 * time.Hour},
		{"provider", 8 * time.Hour},
		{"staff", 12 * time.Hour},
		{"patient", 24 * time.Hour},
		{"unknown", 24 * time.Hour},
		{"  Clinician  ", 8 * time.Hour},
		{"", 24 * time.Hour},
	}

	for _, test := range tests {
		if got := maxLifetimeForRole(test.role); got != test.want {
			t.Errorf("maxLifetimeForRole(%q) = %v, want %v", test.role, got, test.want)
		}
	}
}

// assertNoSessionMutation checks that a rejected command produced no events,
// buffered nothing on the aggregate, and left the version untouched.
func assertNoSessionMutation(t *testing.T, aggregate *SessionAggregate, events []shared.DomainEvent) {
	t.Helper()
	if len(events) != 0 {
		t.Fatalf("Execute() emitted %d events, want 0", len(events))
	}
	if len(aggregate.Events()) != 0 {
		t.Fatalf("aggregate buffered %d events, want 0", len(aggregate.Events()))
	}
	if aggregate.Version != 0 {
		t.Fatalf("aggregate Version = %d, want 0", aggregate.Version)
	}
}
