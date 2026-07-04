package model

import "errors"

var (
	// ErrMissingRegoBundle is returned when PublishPolicyVersionCmd omits the Rego
	// bundle that encodes the ruleset to deploy.
	ErrMissingRegoBundle = errors.New("authorizationpolicy: rego bundle is required")

	// ErrMissingAuthor is returned when PublishPolicyVersionCmd omits the author
	// accountable for the publication.
	ErrMissingAuthor = errors.New("authorizationpolicy: author is required")

	// ErrMissingSubjectAttrs is returned when EvaluateAccessCmd carries no subject
	// attributes to evaluate the policy rules against.
	ErrMissingSubjectAttrs = errors.New("authorizationpolicy: subject attributes are required")

	// ErrMissingResourceRef is returned when EvaluateAccessCmd omits the resource
	// reference the subject is attempting to access.
	ErrMissingResourceRef = errors.New("authorizationpolicy: resource reference is required")

	// ErrMissingAction is returned when EvaluateAccessCmd omits the action the
	// subject wants to perform on the resource.
	ErrMissingAction = errors.New("authorizationpolicy: action is required")

	// ErrMissingCareContext is returned when EvaluateAccessCmd omits the care
	// context the request must be evaluated within.
	ErrMissingCareContext = errors.New("authorizationpolicy: care context is required")

	// ErrDefaultDenyMissing is returned when the ruleset does not default to deny.
	// Invariant: a decision must default to deny when no matching allow rule
	// exists.
	ErrDefaultDenyMissing = errors.New("authorizationpolicy: a decision must default to deny when no matching allow rule exists")

	// ErrPermissionAboveLeastPrivilege is returned when the ruleset grants a role
	// a permission beyond its least-privilege ceiling. Invariant: a role may never
	// be granted a permission above its least-privilege ceiling.
	ErrPermissionAboveLeastPrivilege = errors.New("authorizationpolicy: a role may never be granted a permission above its least-privilege ceiling")

	// ErrNonBinaryDecision is returned when the ruleset can produce a decision that
	// is not exactly one allow or deny outcome. Invariant: every evaluated decision
	// must produce exactly one allow or deny outcome.
	ErrNonBinaryDecision = errors.New("authorizationpolicy: every evaluated decision must produce exactly one allow or deny outcome")

	// ErrPHIScopingMissing is returned when a PHI-bearing resource rule lacks a
	// PHI-scoping predicate. Invariant: PHI-scoping predicates must be present for
	// every PHI-bearing resource rule.
	ErrPHIScopingMissing = errors.New("authorizationpolicy: PHI-scoping predicates must be present for every PHI-bearing resource rule")
)
