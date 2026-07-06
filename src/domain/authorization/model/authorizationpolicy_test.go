package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validAuthorizationPolicyAggregate() *AuthorizationPolicyAggregate {
	return &AuthorizationPolicyAggregate{
		ID: "policy-1",
	}
}

func validPublishPolicyVersionCmd() PublishPolicyVersionCmd {
	return PublishPolicyVersionCmd{
		RegoBundle: "package authz\ndefault allow = false",
		Author:     "author-1",
	}
}

func validEvaluateAccessCmd() EvaluateAccessCmd {
	return EvaluateAccessCmd{
		SubjectAttrs: map[string]string{"role": "clinician"},
		ResourceRef:  "resource-1",
		Action:       "read",
		CareContext:  "encounter-1",
	}
}

func TestPublishPolicyVersionEmitsPolicyPublishedEvent(t *testing.T) {
	agg := validAuthorizationPolicyAggregate()
	cmd := validPublishPolicyVersionCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(PublishPolicyVersionCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(PolicyPublishedEvent)
	if !ok {
		t.Fatalf("event type = %T, want PolicyPublishedEvent", events[0])
	}
	if evt.Type() != PolicyPublishedEventType {
		t.Fatalf("event type = %q, want %q", evt.Type(), PolicyPublishedEventType)
	}
	if evt.Type() != "authz.policy.published" {
		t.Fatalf("event wire name = %q, want authz.policy.published", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.PolicyID != agg.ID {
		t.Fatalf("event policy id = %q, want %q", evt.PolicyID, agg.ID)
	}
	if evt.RegoBundle != cmd.RegoBundle || evt.Author != cmd.Author {
		t.Fatalf("event payload = %#v, want bundle %q author %q", evt, cmd.RegoBundle, cmd.Author)
	}

	// Mutated state.
	if agg.Status != PolicyStatusPublished {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, PolicyStatusPublished)
	}
	if agg.RegoBundle != cmd.RegoBundle {
		t.Fatalf("aggregate rego bundle = %q, want %q", agg.RegoBundle, cmd.RegoBundle)
	}
	if agg.Author != cmd.Author {
		t.Fatalf("aggregate author = %q, want %q", agg.Author, cmd.Author)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestPublishPolicyVersionRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     PublishPolicyVersionCmd
		wantErr error
	}{
		{
			name:    "missing rego bundle",
			cmd:     PublishPolicyVersionCmd{Author: "author-1"},
			wantErr: ErrMissingRegoBundle,
		},
		{
			name:    "missing author",
			cmd:     PublishPolicyVersionCmd{RegoBundle: "package authz"},
			wantErr: ErrMissingAuthor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validAuthorizationPolicyAggregate()

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

func TestPublishPolicyVersionRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mark    func(*AuthorizationPolicyAggregate)
		wantErr error
	}{
		{
			name:    "default deny missing",
			mark:    func(a *AuthorizationPolicyAggregate) { a.DefaultDenyMissing = true },
			wantErr: ErrDefaultDenyMissing,
		},
		{
			name:    "permission above least privilege",
			mark:    func(a *AuthorizationPolicyAggregate) { a.PermissionAboveCeiling = true },
			wantErr: ErrPermissionAboveLeastPrivilege,
		},
		{
			name:    "non binary decision",
			mark:    func(a *AuthorizationPolicyAggregate) { a.NonBinaryDecision = true },
			wantErr: ErrNonBinaryDecision,
		},
		{
			name:    "phi scoping missing",
			mark:    func(a *AuthorizationPolicyAggregate) { a.PHIScopingMissing = true },
			wantErr: ErrPHIScopingMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validAuthorizationPolicyAggregate()
			tt.mark(agg)

			events, err := agg.Execute(validPublishPolicyVersionCmd())
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
			if agg.Status != PolicyStatusDraft {
				t.Fatalf("status = %q, want draft (unchanged)", agg.Status)
			}
		})
	}
}

func TestEvaluateAccessEmitsAccessGrantedEvent(t *testing.T) {
	agg := validAuthorizationPolicyAggregate()
	agg.AccessAllowed = true
	cmd := validEvaluateAccessCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(EvaluateAccessCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(AccessGrantedEvent)
	if !ok {
		t.Fatalf("event type = %T, want AccessGrantedEvent", events[0])
	}
	if evt.Type() != AccessGrantedEventType {
		t.Fatalf("event type = %q, want %q", evt.Type(), AccessGrantedEventType)
	}
	if evt.Type() != "authz.access.granted" {
		t.Fatalf("event wire name = %q, want authz.access.granted", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.PolicyID != agg.ID {
		t.Fatalf("event policy id = %q, want %q", evt.PolicyID, agg.ID)
	}
	if evt.ResourceRef != cmd.ResourceRef || evt.Action != cmd.Action || evt.CareContext != cmd.CareContext {
		t.Fatalf("event payload = %#v", evt)
	}
	if len(evt.SubjectAttrs) != 1 || evt.SubjectAttrs["role"] != "clinician" {
		t.Fatalf("event subject attrs = %#v, want role=clinician", evt.SubjectAttrs)
	}
	if agg.LastDecision != "granted" {
		t.Fatalf("aggregate last decision = %q, want granted", agg.LastDecision)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestEvaluateAccessEmitsAccessDeniedEventOnDefaultDeny(t *testing.T) {
	agg := validAuthorizationPolicyAggregate()
	// AccessAllowed defaults to false: default-deny fallthrough.
	cmd := validEvaluateAccessCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(EvaluateAccessCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(AccessDeniedEvent)
	if !ok {
		t.Fatalf("event type = %T, want AccessDeniedEvent", events[0])
	}
	if evt.Type() != AccessDeniedEventType {
		t.Fatalf("event type = %q, want %q", evt.Type(), AccessDeniedEventType)
	}
	if evt.Type() != "authz.access.denied" {
		t.Fatalf("event wire name = %q, want authz.access.denied", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.PolicyID != agg.ID {
		t.Fatalf("event policy id = %q, want %q", evt.PolicyID, agg.ID)
	}
	if evt.ResourceRef != cmd.ResourceRef || evt.Action != cmd.Action || evt.CareContext != cmd.CareContext {
		t.Fatalf("event payload = %#v", evt)
	}
	if len(evt.SubjectAttrs) != 1 || evt.SubjectAttrs["role"] != "clinician" {
		t.Fatalf("event subject attrs = %#v, want role=clinician", evt.SubjectAttrs)
	}
	if evt.Reason != "no matching allow rule (default deny)" {
		t.Fatalf("event reason = %q", evt.Reason)
	}
	if agg.LastDecision != "denied" {
		t.Fatalf("aggregate last decision = %q, want denied", agg.LastDecision)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestEvaluateAccessRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     EvaluateAccessCmd
		wantErr error
	}{
		{
			name: "missing subject attrs",
			cmd: EvaluateAccessCmd{
				ResourceRef: "resource-1",
				Action:      "read",
				CareContext: "encounter-1",
			},
			wantErr: ErrMissingSubjectAttrs,
		},
		{
			name: "missing resource ref",
			cmd: EvaluateAccessCmd{
				SubjectAttrs: map[string]string{"role": "clinician"},
				Action:       "read",
				CareContext:  "encounter-1",
			},
			wantErr: ErrMissingResourceRef,
		},
		{
			name: "missing action",
			cmd: EvaluateAccessCmd{
				SubjectAttrs: map[string]string{"role": "clinician"},
				ResourceRef:  "resource-1",
				CareContext:  "encounter-1",
			},
			wantErr: ErrMissingAction,
		},
		{
			name: "missing care context",
			cmd: EvaluateAccessCmd{
				SubjectAttrs: map[string]string{"role": "clinician"},
				ResourceRef:  "resource-1",
				Action:       "read",
			},
			wantErr: ErrMissingCareContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validAuthorizationPolicyAggregate()
			agg.AccessAllowed = true // ensure rejection is due to missing field, not deny

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
			if agg.LastDecision != "" {
				t.Fatalf("last decision = %q, want empty (unchanged)", agg.LastDecision)
			}
		})
	}
}

func TestEvaluateAccessRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mark    func(*AuthorizationPolicyAggregate)
		wantErr error
	}{
		{
			name:    "default deny missing",
			mark:    func(a *AuthorizationPolicyAggregate) { a.DefaultDenyMissing = true },
			wantErr: ErrDefaultDenyMissing,
		},
		{
			name:    "permission above least privilege",
			mark:    func(a *AuthorizationPolicyAggregate) { a.PermissionAboveCeiling = true },
			wantErr: ErrPermissionAboveLeastPrivilege,
		},
		{
			name:    "non binary decision",
			mark:    func(a *AuthorizationPolicyAggregate) { a.NonBinaryDecision = true },
			wantErr: ErrNonBinaryDecision,
		},
		{
			name:    "phi scoping missing",
			mark:    func(a *AuthorizationPolicyAggregate) { a.PHIScopingMissing = true },
			wantErr: ErrPHIScopingMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validAuthorizationPolicyAggregate()
			agg.AccessAllowed = true
			tt.mark(agg)

			events, err := agg.Execute(validEvaluateAccessCmd())
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
			if agg.LastDecision != "" {
				t.Fatalf("last decision = %q, want empty (unchanged)", agg.LastDecision)
			}
		})
	}
}

func TestAuthorizationPolicyExecuteRejectsUnknownCommand(t *testing.T) {
	agg := validAuthorizationPolicyAggregate()

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
