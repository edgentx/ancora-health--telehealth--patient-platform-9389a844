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
