package model

import "errors"

var (
	// ErrMissingSubjectAttrs is returned when EvaluateAccessCmd omits the subject
	// attributes.
	ErrMissingSubjectAttrs = errors.New("authorizationpolicy: subject attributes are required")

	// ErrMissingResourceRef is returned when EvaluateAccessCmd omits the resource
	// reference.
	ErrMissingResourceRef = errors.New("authorizationpolicy: resource reference is required")

	// ErrMissingAction is returned when EvaluateAccessCmd omits the action.
	ErrMissingAction = errors.New("authorizationpolicy: action is required")

	// ErrMissingCareContext is returned when EvaluateAccessCmd omits the care
	// context.
	ErrMissingCareContext = errors.New("authorizationpolicy: care context is required")

	// ErrDefaultDenyViolated is returned when the policy would default to allow
	// with no matching allow rule. Invariant: a decision must default to deny when
	// no matching allow rule exists.
	ErrDefaultDenyViolated = errors.New("authorizationpolicy: a decision must default to deny when no matching allow rule exists")

	// ErrPermissionAboveCeiling is returned when a role would be granted a
	// permission above its least-privilege ceiling. Invariant: a role may never be
	// granted a permission above its least-privilege ceiling.
	ErrPermissionAboveCeiling = errors.New("authorizationpolicy: a role may never be granted a permission above its least-privilege ceiling")

	// ErrAmbiguousDecision is returned when an evaluation would produce zero or
	// more than one outcome. Invariant: every evaluated decision must produce
	// exactly one allow or deny outcome.
	ErrAmbiguousDecision = errors.New("authorizationpolicy: every evaluated decision must produce exactly one allow or deny outcome")

	// ErrPHIScopingMissing is returned when a PHI-bearing resource rule lacks a
	// PHI-scoping predicate. Invariant: PHI-scoping predicates must be present for
	// every PHI-bearing resource rule.
	ErrPHIScopingMissing = errors.New("authorizationpolicy: PHI-scoping predicates must be present for every PHI-bearing resource rule")
)
