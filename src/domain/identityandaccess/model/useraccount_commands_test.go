package model

import (
	"errors"
	"testing"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// validRegisterCmd returns a RegisterUserCmd that clears every input guard.
func validRegisterCmd() RegisterUserCmd {
	return RegisterUserCmd{
		Email:    "patient@example.com",
		Password: "correct horse battery staple",
		Role:     "patient",
		TenantId: "tenant-1",
	}
}

func TestRegisterUserEmitsRegisteredEvent(t *testing.T) {
	aggregate := &UserAccountAggregate{ID: "user-123"}
	cmd := RegisterUserCmd{
		Email:    "patient@example.com",
		Password: "correct horse battery staple",
		Role:     "clinician",
		TenantId: "tenant-42",
	}

	events, err := aggregate.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}

	event, ok := events[0].(UserRegisteredEvent)
	if !ok {
		t.Fatalf("event type = %T, want UserRegisteredEvent", events[0])
	}
	if event.Type() != "user.registered" {
		t.Fatalf("event Type() = %q", event.Type())
	}
	if event.AggregateID() != aggregate.ID {
		t.Fatalf("event AggregateID() = %q, want %q", event.AggregateID(), aggregate.ID)
	}
	if event.UserID != aggregate.ID {
		t.Fatalf("event UserID = %q, want %q", event.UserID, aggregate.ID)
	}
	if event.TenantID != cmd.TenantId {
		t.Fatalf("event TenantID = %q, want %q", event.TenantID, cmd.TenantId)
	}
	if event.Email != cmd.Email {
		t.Fatalf("event Email = %q, want %q", event.Email, cmd.Email)
	}
	if event.Role != cmd.Role {
		t.Fatalf("event Role = %q, want %q", event.Role, cmd.Role)
	}
	if event.Route != "/clinician" {
		t.Fatalf("event Route = %q, want %q", event.Route, "/clinician")
	}

	// Mutated aggregate state (via apply).
	if aggregate.TenantID != cmd.TenantId {
		t.Fatalf("aggregate TenantID = %q, want %q", aggregate.TenantID, cmd.TenantId)
	}
	if aggregate.Email != cmd.Email {
		t.Fatalf("aggregate Email = %q, want %q", aggregate.Email, cmd.Email)
	}
	if aggregate.Role != cmd.Role {
		t.Fatalf("aggregate Role = %q, want %q", aggregate.Role, cmd.Role)
	}
	if !aggregate.EmailRegistered {
		t.Fatal("aggregate EmailRegistered = false, want true")
	}
	if aggregate.Version != 1 {
		t.Fatalf("aggregate Version = %d, want 1", aggregate.Version)
	}
	if len(aggregate.Events()) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(aggregate.Events()))
	}
	if aggregate.GetVersion() != 1 {
		t.Fatalf("aggregate GetVersion() = %d, want 1", aggregate.GetVersion())
	}
}

func TestRegisterUserRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*RegisterUserCmd)
		wantErr error
	}{
		{
			name:    "missing email",
			mutate:  func(c *RegisterUserCmd) { c.Email = "   " },
			wantErr: ErrMissingEmail,
		},
		{
			name:    "missing password",
			mutate:  func(c *RegisterUserCmd) { c.Password = "" },
			wantErr: ErrMissingPassword,
		},
		{
			name:    "missing role",
			mutate:  func(c *RegisterUserCmd) { c.Role = " " },
			wantErr: ErrMissingRole,
		},
		{
			name:    "missing tenant",
			mutate:  func(c *RegisterUserCmd) { c.TenantId = "" },
			wantErr: ErrMissingTenant,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := &UserAccountAggregate{ID: "user-123"}
			cmd := validRegisterCmd()
			test.mutate(&cmd)

			events, err := aggregate.Execute(cmd)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoMutation(t, aggregate, events)
		})
	}
}

func TestRegisterUserRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name      string
		aggregate UserAccountAggregate
		wantErr   error
	}{
		{
			name:      "email already registered",
			aggregate: UserAccountAggregate{ID: "user-123", EmailRegistered: true},
			wantErr:   ErrEmailAlreadyRegistered,
		},
		{
			name:      "account locked",
			aggregate: UserAccountAggregate{ID: "user-123", LockedUntil: time.Now().Add(time.Hour)},
			wantErr:   ErrAccountLocked,
		},
		{
			name:      "reset token consumed",
			aggregate: UserAccountAggregate{ID: "user-123", ResetTokenConsumed: true},
			wantErr:   ErrResetTokenInvalid,
		},
		{
			name:      "reset token expired",
			aggregate: UserAccountAggregate{ID: "user-123", ResetTokenExpiresAt: time.Now().Add(-time.Minute)},
			wantErr:   ErrResetTokenInvalid,
		},
		{
			name:      "second factor missing",
			aggregate: UserAccountAggregate{ID: "user-123", MFAEnrolled: true},
			wantErr:   ErrSecondFactorRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := test.aggregate

			events, err := aggregate.Execute(validRegisterCmd())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoMutation(t, &aggregate, events)
		})
	}
}

func TestLockAccountEmitsLockedEvent(t *testing.T) {
	aggregate := &UserAccountAggregate{ID: "user-123"}
	cmd := LockAccountCmd{AccountId: "user-123", Reason: "failed-attempt threshold exceeded"}

	before := time.Now()
	events, err := aggregate.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}

	event, ok := events[0].(UserAccountLockedEvent)
	if !ok {
		t.Fatalf("event type = %T, want UserAccountLockedEvent", events[0])
	}
	if event.Type() != "user.account.locked" {
		t.Fatalf("event Type() = %q", event.Type())
	}
	if event.AggregateID() != aggregate.ID {
		t.Fatalf("event AggregateID() = %q, want %q", event.AggregateID(), aggregate.ID)
	}
	if event.UserID != aggregate.ID {
		t.Fatalf("event UserID = %q, want %q", event.UserID, aggregate.ID)
	}
	if event.Reason != cmd.Reason {
		t.Fatalf("event Reason = %q, want %q", event.Reason, cmd.Reason)
	}
	wantMin := before.Add(lockoutWindow)
	if event.LockedUntil.Before(wantMin) || event.LockedUntil.After(time.Now().Add(lockoutWindow).Add(time.Second)) {
		t.Fatalf("event LockedUntil = %v, want ~now+%v", event.LockedUntil, lockoutWindow)
	}
	if !aggregate.LockedUntil.Equal(event.LockedUntil) {
		t.Fatalf("aggregate LockedUntil = %v, want %v", aggregate.LockedUntil, event.LockedUntil)
	}
	if aggregate.Version != 1 {
		t.Fatalf("aggregate Version = %d, want 1", aggregate.Version)
	}
	if len(aggregate.Events()) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(aggregate.Events()))
	}
}

func TestLockAccountRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		cmd     LockAccountCmd
		wantErr error
	}{
		{
			name:    "missing account id",
			cmd:     LockAccountCmd{AccountId: "  ", Reason: "compromise"},
			wantErr: ErrMissingAccountID,
		},
		{
			name:    "missing reason",
			cmd:     LockAccountCmd{AccountId: "user-123", Reason: ""},
			wantErr: ErrMissingReason,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := &UserAccountAggregate{ID: "user-123"}

			events, err := aggregate.Execute(test.cmd)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoMutation(t, aggregate, events)
		})
	}
}

func TestLockAccountRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name      string
		aggregate UserAccountAggregate
		wantErr   error
	}{
		{
			name:      "email already registered",
			aggregate: UserAccountAggregate{ID: "user-123", EmailRegistered: true},
			wantErr:   ErrEmailAlreadyRegistered,
		},
		{
			name:      "account locked",
			aggregate: UserAccountAggregate{ID: "user-123", LockedUntil: time.Now().Add(time.Hour)},
			wantErr:   ErrAccountLocked,
		},
		{
			name:      "reset token consumed",
			aggregate: UserAccountAggregate{ID: "user-123", ResetTokenConsumed: true},
			wantErr:   ErrResetTokenInvalid,
		},
		{
			name:      "second factor missing",
			aggregate: UserAccountAggregate{ID: "user-123", MFAEnrolled: true},
			wantErr:   ErrSecondFactorRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := test.aggregate

			events, err := aggregate.Execute(LockAccountCmd{AccountId: "user-123", Reason: "compromise"})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoMutation(t, &aggregate, events)
		})
	}
}

func TestAuthenticateUserSucceedsOnMatchingCredential(t *testing.T) {
	aggregate := &UserAccountAggregate{
		ID:                  "user-123",
		TenantID:            "tenant-9",
		CredentialVerified:  true,
		FailedLoginAttempts: 3,
	}
	cmd := AuthenticateUserCmd{Email: "patient@example.com", Password: "pw", MfaCode: "123456"}

	events, err := aggregate.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}

	event, ok := events[0].(UserAuthenticatedEvent)
	if !ok {
		t.Fatalf("event type = %T, want UserAuthenticatedEvent", events[0])
	}
	if event.Type() != "user.authenticated" {
		t.Fatalf("event Type() = %q", event.Type())
	}
	if event.AggregateID() != aggregate.ID {
		t.Fatalf("event AggregateID() = %q, want %q", event.AggregateID(), aggregate.ID)
	}
	if event.UserID != aggregate.ID {
		t.Fatalf("event UserID = %q, want %q", event.UserID, aggregate.ID)
	}
	if event.TenantID != "tenant-9" {
		t.Fatalf("event TenantID = %q, want %q", event.TenantID, "tenant-9")
	}
	if event.Email != cmd.Email {
		t.Fatalf("event Email = %q, want %q", event.Email, cmd.Email)
	}
	if aggregate.FailedLoginAttempts != 0 {
		t.Fatalf("aggregate FailedLoginAttempts = %d, want 0", aggregate.FailedLoginAttempts)
	}
	if aggregate.Version != 1 {
		t.Fatalf("aggregate Version = %d, want 1", aggregate.Version)
	}
	if len(aggregate.Events()) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(aggregate.Events()))
	}
}

func TestAuthenticateUserEmitsLoginFailedOnBadCredential(t *testing.T) {
	aggregate := &UserAccountAggregate{
		ID:                  "user-123",
		CredentialVerified:  false,
		FailedLoginAttempts: 2,
	}
	cmd := AuthenticateUserCmd{Email: "patient@example.com", Password: "wrong", MfaCode: "123456"}

	events, err := aggregate.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Execute() emitted %d events, want 1", len(events))
	}

	event, ok := events[0].(UserLoginFailedEvent)
	if !ok {
		t.Fatalf("event type = %T, want UserLoginFailedEvent", events[0])
	}
	if event.Type() != "user.login.failed" {
		t.Fatalf("event Type() = %q", event.Type())
	}
	if event.AggregateID() != aggregate.ID {
		t.Fatalf("event AggregateID() = %q, want %q", event.AggregateID(), aggregate.ID)
	}
	if event.UserID != aggregate.ID {
		t.Fatalf("event UserID = %q, want %q", event.UserID, aggregate.ID)
	}
	if event.Email != cmd.Email {
		t.Fatalf("event Email = %q, want %q", event.Email, cmd.Email)
	}
	if event.FailedAttempts != 3 {
		t.Fatalf("event FailedAttempts = %d, want 3", event.FailedAttempts)
	}
	if aggregate.FailedLoginAttempts != 3 {
		t.Fatalf("aggregate FailedLoginAttempts = %d, want 3", aggregate.FailedLoginAttempts)
	}
	if aggregate.Version != 1 {
		t.Fatalf("aggregate Version = %d, want 1", aggregate.Version)
	}
	if len(aggregate.Events()) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(aggregate.Events()))
	}
}

func TestAuthenticateUserRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		cmd     AuthenticateUserCmd
		wantErr error
	}{
		{
			name:    "missing email",
			cmd:     AuthenticateUserCmd{Email: "", Password: "pw", MfaCode: "123456"},
			wantErr: ErrMissingEmail,
		},
		{
			name:    "missing password",
			cmd:     AuthenticateUserCmd{Email: "patient@example.com", Password: "   ", MfaCode: "123456"},
			wantErr: ErrMissingPassword,
		},
		{
			name:    "missing mfa code",
			cmd:     AuthenticateUserCmd{Email: "patient@example.com", Password: "pw", MfaCode: ""},
			wantErr: ErrMissingMfaCode,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := &UserAccountAggregate{ID: "user-123", CredentialVerified: true}

			events, err := aggregate.Execute(test.cmd)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoMutation(t, aggregate, events)
		})
	}
}

func TestAuthenticateUserRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name      string
		aggregate UserAccountAggregate
		wantErr   error
	}{
		{
			name:      "email already registered",
			aggregate: UserAccountAggregate{ID: "user-123", EmailRegistered: true},
			wantErr:   ErrEmailAlreadyRegistered,
		},
		{
			name:      "account locked",
			aggregate: UserAccountAggregate{ID: "user-123", LockedUntil: time.Now().Add(time.Hour)},
			wantErr:   ErrAccountLocked,
		},
		{
			name:      "reset token consumed",
			aggregate: UserAccountAggregate{ID: "user-123", ResetTokenConsumed: true},
			wantErr:   ErrResetTokenInvalid,
		},
		{
			name:      "second factor missing",
			aggregate: UserAccountAggregate{ID: "user-123", MFAEnrolled: true},
			wantErr:   ErrSecondFactorRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := test.aggregate

			events, err := aggregate.Execute(AuthenticateUserCmd{Email: "patient@example.com", Password: "pw", MfaCode: "123456"})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			assertNoMutation(t, &aggregate, events)
		})
	}
}

func TestInitiatePasswordResetRejectsMissingEmail(t *testing.T) {
	aggregate := &UserAccountAggregate{ID: "user-123"}

	events, err := aggregate.Execute(InitiatePasswordResetCmd{Email: "   "})
	if !errors.Is(err, ErrMissingEmail) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrMissingEmail)
	}
	assertNoMutation(t, aggregate, events)
}

func TestUserAccountExecuteRejectsUnknownCommand(t *testing.T) {
	aggregate := &UserAccountAggregate{ID: "user-123"}

	type bogusCmd struct{}
	events, err := aggregate.Execute(bogusCmd{})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("Execute() error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	assertNoMutation(t, aggregate, events)
}

func TestRouteForRole(t *testing.T) {
	tests := []struct {
		role string
		want string
	}{
		{"clinician", "/clinician"},
		{"provider", "/clinician"},
		{"Clinician", "/clinician"},
		{"admin", "/admin"},
		{"administrator", "/admin"},
		{"  ADMIN  ", "/admin"},
		{"staff", "/staff"},
		{"patient", "/patient"},
		{"unknown", "/patient"},
		{"", "/patient"},
	}

	for _, test := range tests {
		if got := routeForRole(test.role); got != test.want {
			t.Errorf("routeForRole(%q) = %q, want %q", test.role, got, test.want)
		}
	}
}

func TestNewResetTokenIsUnpredictable(t *testing.T) {
	a := newResetToken()
	b := newResetToken()
	if a == "" || b == "" {
		t.Fatal("newResetToken() returned an empty token")
	}
	if a == b {
		t.Fatalf("newResetToken() returned identical tokens %q", a)
	}
	// 32 random bytes hex-encoded is 64 characters.
	if len(a) != 64 {
		t.Fatalf("newResetToken() length = %d, want 64", len(a))
	}
}

// assertNoMutation checks that a rejected command produced no events, buffered
// nothing on the aggregate, and left the version untouched.
func assertNoMutation(t *testing.T, aggregate *UserAccountAggregate, events []shared.DomainEvent) {
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
