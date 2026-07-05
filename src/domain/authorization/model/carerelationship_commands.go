package model

// EstablishCareRelationshipCmd requests that a care-team-to-patient grant be
// created, authorizing a provider to access a patient's PHI within a clinic.
// It carries the provider, patient, and clinic that scope the grant.
//
// Establishing the relationship is the act that puts an active care grant into
// force. The command must satisfy the care-relationship invariants before it
// can succeed: a provider may only access a patient's PHI when an active care
// relationship exists, a care relationship must be revoked when the care
// episode ends, and a relationship cannot be self-asserted by the accessing
// party without a governing grant. ProviderID, PatientID, and ClinicID scope
// the grant and are all mandatory.
type EstablishCareRelationshipCmd struct {
	// ProviderID identifies the care-team member being granted access to the
	// patient's PHI.
	ProviderID string
	// PatientID identifies the patient whose PHI the grant authorizes access to.
	PatientID string
	// ClinicID identifies the clinic the care relationship is scoped within.
	ClinicID string
}

// RevokeCareRelationshipCmd requests that an active care-team-to-patient grant be
// ended, withdrawing the provider's authorization to access the patient's PHI.
// It carries the relationship being revoked and the reason it is being ended.
//
// Revoking is the act that takes an in-force care grant out of force — for
// example when the care episode ends. The command must satisfy the
// care-relationship invariants before it can succeed: a provider may only access
// a patient's PHI when an active care relationship exists, a care relationship
// must be revoked when the care episode ends, and a relationship cannot be
// self-asserted by the accessing party without a governing grant. RelationshipID
// and Reason are both mandatory.
type RevokeCareRelationshipCmd struct {
	// RelationshipID identifies the care relationship whose grant is being
	// revoked.
	RelationshipID string
	// Reason records why the care relationship is being ended.
	Reason string
}
