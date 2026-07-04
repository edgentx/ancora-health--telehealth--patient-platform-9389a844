// Package model holds the aggregates for the authorization bounded context.
// AuthorizationPolicyAggregate governs access-control policy decisions;
// PublishPolicyVersionCmd deploys a new immutable ruleset version.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PolicyStatus is the lifecycle state of an authorization policy. The zero value
// is a policy that has not yet published a ruleset version, which is what
// PublishPolicyVersionCmd acts on.
type PolicyStatus string

const (
	// PolicyStatusDraft is a policy that has not yet published a ruleset version.
	// It is the zero value, so a freshly constructed aggregate is a draft.
	PolicyStatusDraft PolicyStatus = ""
	// PolicyStatusPublished is a policy that has a published, in-force ruleset
	// version.
	PolicyStatusPublished PolicyStatus = "published"
)

// AuthorizationPolicyAggregate is the authorization aggregate that governs
// access-control policy decisions. It embeds shared.AggregateRoot for version
// tracking and event buffering, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the Rego bundle of the in-force ruleset, the author who
// published it, and the flags describing whether the candidate ruleset violates
// one of the policy invariants.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type AuthorizationPolicyAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the policy's lifecycle state.
	Status PolicyStatus

	// RegoBundle is the immutable ruleset of the in-force version. It is empty
	// until a version is published.
	RegoBundle string

	// Author is who published the in-force ruleset version. It is empty until a
	// version is published.
	Author string

	// DefaultDenyMissing reports that the candidate ruleset does not default to
	// deny. Invariant: a decision must default to deny when no matching allow rule
	// exists.
	DefaultDenyMissing bool

	// PermissionAboveCeiling reports that the candidate ruleset grants a role a
	// permission beyond its least-privilege ceiling. Invariant: a role may never
	// be granted a permission above its least-privilege ceiling.
	PermissionAboveCeiling bool

	// NonBinaryDecision reports that the candidate ruleset can produce a decision
	// that is not exactly one allow or deny outcome. Invariant: every evaluated
	// decision must produce exactly one allow or deny outcome.
	NonBinaryDecision bool

	// PHIScopingMissing reports that a PHI-bearing resource rule lacks a
	// PHI-scoping predicate. Invariant: PHI-scoping predicates must be present for
	// every PHI-bearing resource rule.
	PHIScopingMissing bool

	// AccessAllowed is the settled verdict of the in-force ruleset for the request
	// under evaluation: true when a matching allow rule was found, false otherwise.
	// The policy engine computes it upstream, so EvaluateAccessCmd reads it rather
	// than re-running the Rego evaluation in the domain layer. Its zero value is
	// deny, which realizes the policy's default-deny stance.
	AccessAllowed bool

	// LastDecision records the outcome of the most recent access evaluation,
	// mirroring the emitted event type ("granted" or "denied"). It is empty until
	// the first EvaluateAccessCmd is handled.
	LastDecision string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *AuthorizationPolicyAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case PublishPolicyVersionCmd:
		return a.publishPolicyVersion(c)
	case EvaluateAccessCmd:
		return a.evaluateAccess(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// publishPolicyVersion handles PublishPolicyVersionCmd: it validates the command
// input, enforces the policy invariants, then emits a PolicyPublishedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the Rego bundle and author must both be present.
//   - Default deny: a decision must default to deny when no matching allow rule
//     exists.
//   - Least privilege: a role may never be granted a permission above its
//     least-privilege ceiling.
//   - Binary decision: every evaluated decision must produce exactly one allow or
//     deny outcome.
//   - PHI scoping: PHI-scoping predicates must be present for every PHI-bearing
//     resource rule.
func (a *AuthorizationPolicyAggregate) publishPolicyVersion(cmd PublishPolicyVersionCmd) ([]shared.DomainEvent, error) {
	if cmd.RegoBundle == "" {
		return nil, ErrMissingRegoBundle
	}
	if cmd.Author == "" {
		return nil, ErrMissingAuthor
	}

	// Invariant: a decision must default to deny when no matching allow rule
	// exists.
	if a.DefaultDenyMissing {
		return nil, ErrDefaultDenyMissing
	}

	// Invariant: a role may never be granted a permission above its
	// least-privilege ceiling.
	if a.PermissionAboveCeiling {
		return nil, ErrPermissionAboveLeastPrivilege
	}

	// Invariant: every evaluated decision must produce exactly one allow or deny
	// outcome.
	if a.NonBinaryDecision {
		return nil, ErrNonBinaryDecision
	}

	// Invariant: PHI-scoping predicates must be present for every PHI-bearing
	// resource rule.
	if a.PHIScopingMissing {
		return nil, ErrPHIScopingMissing
	}

	evt := PolicyPublishedEvent{
		PolicyID:   a.ID,
		RegoBundle: cmd.RegoBundle,
		Author:     cmd.Author,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *AuthorizationPolicyAggregate) apply(evt PolicyPublishedEvent) {
	a.Status = PolicyStatusPublished
	a.RegoBundle = evt.RegoBundle
	a.Author = evt.Author
}

// evaluateAccess handles EvaluateAccessCmd: it validates the command input,
// enforces the policy invariants, then renders exactly one allow/deny decision
// and emits the matching event.
//
// A deny is not a domain error but a legitimate outcome, so after the guards
// pass the handler always succeeds — emitting AccessGrantedEvent when the
// in-force ruleset matched an allow rule, and AccessDeniedEvent otherwise. The
// deny branch realizes the policy's default-deny stance: with no matching allow
// rule the request falls through to denial rather than surfacing as an error.
//
// The guards enforce, in order:
//
//   - Completeness: subject attributes, resource reference, action, and care
//     context must all be present.
//   - Default deny: a decision must default to deny when no matching allow rule
//     exists.
//   - Least privilege: a role may never be granted a permission above its
//     least-privilege ceiling.
//   - Binary decision: every evaluated decision must produce exactly one allow or
//     deny outcome.
//   - PHI scoping: PHI-scoping predicates must be present for every PHI-bearing
//     resource rule.
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
	if a.DefaultDenyMissing {
		return nil, ErrDefaultDenyMissing
	}

	// Invariant: a role may never be granted a permission above its
	// least-privilege ceiling.
	if a.PermissionAboveCeiling {
		return nil, ErrPermissionAboveLeastPrivilege
	}

	// Invariant: every evaluated decision must produce exactly one allow or deny
	// outcome.
	if a.NonBinaryDecision {
		return nil, ErrNonBinaryDecision
	}

	// Invariant: PHI-scoping predicates must be present for every PHI-bearing
	// resource rule.
	if a.PHIScopingMissing {
		return nil, ErrPHIScopingMissing
	}

	// A matching allow rule grants access; anything else falls through to the
	// policy's default-deny stance.
	if !a.AccessAllowed {
		evt := AccessDeniedEvent{
			PolicyID:     a.ID,
			SubjectAttrs: cmd.SubjectAttrs,
			ResourceRef:  cmd.ResourceRef,
			Action:       cmd.Action,
			CareContext:  cmd.CareContext,
			Reason:       "no matching allow rule (default deny)",
		}

		a.applyDenied(evt)
		a.AddEvent(evt)
		a.Version++

		return []shared.DomainEvent{evt}, nil
	}

	evt := AccessGrantedEvent{
		PolicyID:     a.ID,
		SubjectAttrs: cmd.SubjectAttrs,
		ResourceRef:  cmd.ResourceRef,
		Action:       cmd.Action,
		CareContext:  cmd.CareContext,
	}

	a.applyGranted(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyGranted mutates aggregate state from an AccessGrantedEvent, recording that
// the most recent evaluation resolved to allow. Keeping mutation here lets the
// same event drive both command handling and future replay when rehydrating from
// the store.
func (a *AuthorizationPolicyAggregate) applyGranted(evt AccessGrantedEvent) {
	a.LastDecision = "granted"
}

// applyDenied mutates aggregate state from an AccessDeniedEvent, recording that
// the most recent evaluation resolved to deny.
func (a *AuthorizationPolicyAggregate) applyDenied(evt AccessDeniedEvent) {
	a.LastDecision = "denied"
}
