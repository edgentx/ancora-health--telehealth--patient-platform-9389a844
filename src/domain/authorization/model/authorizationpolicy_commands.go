package model

// EvaluateAccessCmd requests an access-control decision for a subject acting on
// a resource within a care context. Evaluating the policy renders exactly one
// allow-or-deny outcome for the subject-resource pair.
//
// Rendering a decision is the act that turns a policy into an answer: the
// decision defaults to deny unless a matching allow rule exists, a role may
// never be granted a permission above its least-privilege ceiling, every
// evaluated decision must produce exactly one allow or deny outcome, and every
// PHI-bearing resource rule must carry a PHI-scoping predicate. SubjectAttrs
// describe the requesting subject, ResourceRef identifies the target resource,
// Action names the operation being attempted, and CareContext scopes the
// request to a care relationship. All four are mandatory.
type EvaluateAccessCmd struct {
	// SubjectAttrs are the attributes of the subject requesting access (for
	// example role, tenant and clearance). At least one attribute is required.
	SubjectAttrs map[string]string
	// ResourceRef identifies the resource the subject is attempting to access.
	ResourceRef string
	// Action names the operation the subject is attempting on the resource.
	Action string
	// CareContext scopes the request to the care relationship it is evaluated
	// within.
	CareContext string
}
