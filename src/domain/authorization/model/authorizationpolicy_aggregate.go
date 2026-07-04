// Package model holds the aggregates for the authorization bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AccessDecision is the outcome of an evaluated access request. The zero value
// is an undecided policy that has not yet rendered a decision.
type AccessDecision string

const (
	// AccessDecisionUndecided is a policy that has not yet rendered a decision. It
	// is the zero value, so a freshly constructed aggregate is undecided.
	AccessDecisionUndecided AccessDecision = ""
	// AccessDecisionGranted marks the last evaluation as having allowed access.
	AccessDecisionGranted AccessDecision = "granted"
	// AccessDecisionDenied marks the last evaluation as having denied access.
	AccessDecisionDenied AccessDecision = "denied"
)

// AuthorizationPolicyAggregate is the authorization aggregate that governs
// access-control policy decisions. It embeds shared.AggregateRoot for version
// tracking and event buffering, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: whether a
// matching allow rule renders the access grant, the last decision it rendered
// and what that decision was rendered for, and the flags describing the policy
// misconfigurations the guards reject.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type AuthorizationPolicyAggregate struct {
	shared.AggregateRoot
	ID string

	// AccessGranted reports that a matching allow rule, within the subject's
	// least-privilege ceiling, renders the request an allow. Its zero value is
	// false, so a policy with no matching allow rule defaults to deny.
	AccessGranted bool

	// LastDecision is the outcome of the most recent evaluation, and
	// EvaluatedResourceRef / EvaluatedAction record what it was rendered for. They
	// are the zero value until the first EvaluateAccessCmd is executed.
	LastDecision         AccessDecision
	EvaluatedResourceRef string
	EvaluatedAction      string

	// DefaultsToAllow reports that the policy would default to allow with no
	// matching allow rule. Invariant: a decision must default to deny when no
	// matching allow rule exists.
	DefaultsToAllow bool

	// PermissionAboveCeiling reports that a role would be granted a permission
	// above its least-privilege ceiling. Invariant: a role may never be granted a
	// permission above its least-privilege ceiling.
	PermissionAboveCeiling bool

	// AmbiguousDecision reports that an evaluation would produce zero or more than
	// one outcome. Invariant: every evaluated decision must produce exactly one
	// allow or deny outcome.
	AmbiguousDecision bool

	// PHIScopingMissing reports that a PHI-bearing resource rule lacks a
	// PHI-scoping predicate. Invariant: PHI-scoping predicates must be present for
	// every PHI-bearing resource rule.
	PHIScopingMissing bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *AuthorizationPolicyAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case EvaluateAccessCmd:
		return a.evaluateAccess(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// evaluateAccess handles EvaluateAccessCmd: it validates the command input,
// enforces the policy invariants, then renders exactly one allow-or-deny
// decision and emits the corresponding event, buffering it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the subject attributes, resource reference, action and care
//     context must all be present.
//   - Default deny: a decision must default to deny when no matching allow rule
//     exists.
//   - Least privilege: a role may never be granted a permission above its
//     least-privilege ceiling.
//   - Single outcome: every evaluated decision must produce exactly one allow or
//     deny outcome.
//   - PHI scoping: PHI-scoping predicates must be present for every PHI-bearing
//     resource rule.
//
// Once the guards pass, the decision is rendered: a matching allow rule within
// the least-privilege ceiling (AccessGranted) yields an AccessGrantedEvent,
// otherwise the policy defaults to deny and yields an AccessDeniedEvent. Exactly
// one outcome event is ever emitted for an evaluation.
func (a *AuthorizationPolicyAggregate) evaluateAccess(cmd EvaluateAccessCmd) ([]shared.DomainEvent, error) {
	if len(cmd.SubjectAttrs) == 0 {
		return nil, ErrMissingSubjectAttrs
	}
	if cmd.ResourceRef == "" {
		return nil, ErrMissingResourceRef
	}
	if cmd.Action == "" {
		return nil, ErrMissingAction
	}
	if cmd.CareContext == "" {
		return nil, ErrMissingCareContext
	}

	// Invariant: a decision must default to deny when no matching allow rule
	// exists.
	if a.DefaultsToAllow {
		return nil, ErrDefaultDenyViolated
	}

	// Invariant: a role may never be granted a permission above its
	// least-privilege ceiling.
	if a.PermissionAboveCeiling {
		return nil, ErrPermissionAboveCeiling
	}

	// Invariant: every evaluated decision must produce exactly one allow or deny
	// outcome.
	if a.AmbiguousDecision {
		return nil, ErrAmbiguousDecision
	}

	// Invariant: PHI-scoping predicates must be present for every PHI-bearing
	// resource rule.
	if a.PHIScopingMissing {
		return nil, ErrPHIScopingMissing
	}

	// Render exactly one outcome. A matching allow rule within the least-privilege
	// ceiling grants; otherwise the policy defaults to deny.
	if a.AccessGranted {
		evt := AccessGrantedEvent{
			PolicyID:    a.ID,
			ResourceRef: cmd.ResourceRef,
			Action:      cmd.Action,
			CareContext: cmd.CareContext,
		}
		a.applyGranted(evt)
		a.AddEvent(evt)
		a.Version++
		return []shared.DomainEvent{evt}, nil
	}

	evt := AccessDeniedEvent{
		PolicyID:    a.ID,
		ResourceRef: cmd.ResourceRef,
		Action:      cmd.Action,
		CareContext: cmd.CareContext,
	}
	a.applyDenied(evt)
	a.AddEvent(evt)
	a.Version++
	return []shared.DomainEvent{evt}, nil
}

// applyGranted mutates aggregate state from an AccessGrantedEvent. Like the
// deny counterpart it is the single place grant-decision state changes, so it
// serves both command handling and future event replay when rehydrating the
// aggregate from the store.
func (a *AuthorizationPolicyAggregate) applyGranted(evt AccessGrantedEvent) {
	a.LastDecision = AccessDecisionGranted
	a.EvaluatedResourceRef = evt.ResourceRef
	a.EvaluatedAction = evt.Action
}

// applyDenied mutates aggregate state from an AccessDeniedEvent. Like the grant
// counterpart it is the single place deny-decision state changes, so it serves
// both command handling and future event replay when rehydrating from the store.
func (a *AuthorizationPolicyAggregate) applyDenied(evt AccessDeniedEvent) {
	a.LastDecision = AccessDecisionDenied
	a.EvaluatedResourceRef = evt.ResourceRef
	a.EvaluatedAction = evt.Action
}
