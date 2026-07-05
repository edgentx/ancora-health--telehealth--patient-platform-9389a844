package model

import (
	"errors"
	"testing"
	"time"
)

func TestInitiatePasswordResetEmitsRequestedEvent(t *testing.T) {
	aggregate := &UserAccountAggregate{ID: "user-123"}
	email := "patient@example.com"
	startedAt := time.Now()

	events, err := aggregate.Execute(InitiatePasswordResetCmd{Email: email})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}

	event, ok := events[0].(UserPasswordResetRequestedEvent)
	if !ok {
		t.Fatalf("event type = %T, want UserPasswordResetRequestedEvent", events[0])
	}
	if event.Type() != "user.password.reset.requested" {
		t.Fatalf("event Type() = %q", event.Type())
	}
	if event.AggregateID() != aggregate.ID {
		t.Fatalf("event AggregateID() = %q, want %q", event.AggregateID(), aggregate.ID)
	}
	if event.UserID != aggregate.ID {
		t.Fatalf("event UserID = %q, want %q", event.UserID, aggregate.ID)
	}
	if event.Email != email {
		t.Fatalf("event Email = %q, want %q", event.Email, email)
	}
	if event.Token == "" {
		t.Fatal("event Token is empty")
	}
	if !event.ExpiresAt.After(startedAt) {
		t.Fatalf("event ExpiresAt = %v, want after %v", event.ExpiresAt, startedAt)
	}
	if event.ExpiresAt.After(startedAt.Add(resetTokenWindow).Add(time.Second)) {
		t.Fatalf("event ExpiresAt = %v, want within reset token window", event.ExpiresAt)
	}
	if aggregate.ResetTokenConsumed {
		t.Fatal("aggregate ResetTokenConsumed = true, want false")
	}
	if !aggregate.ResetTokenExpiresAt.Equal(event.ExpiresAt) {
		t.Fatalf("aggregate ResetTokenExpiresAt = %v, want %v", aggregate.ResetTokenExpiresAt, event.ExpiresAt)
	}
	if aggregate.Version != 1 {
		t.Fatalf("aggregate Version = %d, want 1", aggregate.Version)
	}
	if len(aggregate.Events()) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(aggregate.Events()))
	}
}

func TestInitiatePasswordResetAllowsExpiredPriorToken(t *testing.T) {
	aggregate := &UserAccountAggregate{
		ID:                  "user-123",
		ResetTokenExpiresAt: time.Now().Add(-time.Minute),
	}

	events, err := aggregate.Execute(InitiatePasswordResetCmd{Email: "patient@example.com"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}
}

func TestInitiatePasswordResetRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name      string
		aggregate UserAccountAggregate
		wantErr   error
	}{
		{
			name: "email already claimed by active account",
			aggregate: UserAccountAggregate{
				ID:              "user-123",
				EmailRegistered: true,
			},
			wantErr: ErrEmailAlreadyRegistered,
		},
		{
			name: "credential stuffing lockout active",
			aggregate: UserAccountAggregate{
				ID:          "user-123",
				LockedUntil: time.Now().Add(time.Hour),
			},
			wantErr: ErrAccountLocked,
		},
		{
			name: "reset token already consumed",
			aggregate: UserAccountAggregate{
				ID:                 "user-123",
				ResetTokenConsumed: true,
			},
			wantErr: ErrResetTokenInvalid,
		},
		{
			name: "reset token still unexpired",
			aggregate: UserAccountAggregate{
				ID:                  "user-123",
				ResetTokenExpiresAt: time.Now().Add(time.Minute),
			},
			wantErr: ErrResetTokenInvalid,
		},
		{
			name: "mfa second factor missing",
			aggregate: UserAccountAggregate{
				ID:          "user-123",
				MFAEnrolled: true,
			},
			wantErr: ErrSecondFactorRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := test.aggregate

			events, err := aggregate.Execute(InitiatePasswordResetCmd{Email: "patient@example.com"})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			if len(events) != 0 {
				t.Fatalf("Execute() emitted %d events, want 0", len(events))
			}
			if len(aggregate.Events()) != 0 {
				t.Fatalf("aggregate buffered %d events, want 0", len(aggregate.Events()))
			}
			if aggregate.Version != 0 {
				t.Fatalf("aggregate Version = %d, want 0", aggregate.Version)
			}
		})
	}
}
