package model

// PublishPolicyVersionCmd requests that a new immutable ruleset version be
// deployed for the authorization policy. It carries the Rego bundle that
// encodes the ruleset and the author accountable for the publication.
//
// Publishing a version is the act that puts a new ruleset into force. The
// bundle must satisfy the policy invariants before it can be published: a
// decision must default to deny when no matching allow rule exists, a role may
// never be granted a permission above its least-privilege ceiling, every
// evaluated decision must produce exactly one allow or deny outcome, and
// PHI-scoping predicates must be present for every PHI-bearing resource rule.
// RegoBundle carries the ruleset and Author identifies who published it; both
// are mandatory.
type PublishPolicyVersionCmd struct {
	// RegoBundle is the immutable ruleset, encoded as a Rego policy bundle, to
	// deploy as the new version.
	RegoBundle string
	// Author identifies who is publishing the new ruleset version.
	Author string
}

// EvaluateAccessCmd requests an allow/deny decision for a single subject-resource
// pair against the in-force policy. It renders exactly one outcome: the subject,
// described by SubjectAttrs, is either granted or denied the requested Action on
// the resource identified by ResourceRef, evaluated within CareContext.
//
// The decision is not itself a domain error — a deny is a legitimate outcome
// emitted as an AccessDeniedEvent, mirroring the policy's default-deny stance. A
// command is only rejected with a domain error when it is malformed (a missing
// attribute, resource, action, or care context) or when the in-force ruleset
// violates a policy invariant: a decision must default to deny when no matching
// allow rule exists, a role may never be granted a permission above its
// least-privilege ceiling, every evaluated decision must produce exactly one
// allow or deny outcome, and PHI-scoping predicates must be present for every
// PHI-bearing resource rule.
type EvaluateAccessCmd struct {
	// SubjectAttrs are the attributes of the subject requesting access (e.g. its
	// role, tenant, and clearances) that the policy rules are evaluated against.
	// At least one attribute must be present.
	SubjectAttrs map[string]string
	// ResourceRef identifies the resource the subject is attempting to access.
	ResourceRef string
	// Action is the operation the subject wants to perform on the resource (e.g.
	// "read", "write", "delete").
	Action string
	// CareContext is the clinical relationship the request is evaluated within
	// (e.g. the treating encounter or care-team scope) that gates PHI access.
	CareContext string
}
