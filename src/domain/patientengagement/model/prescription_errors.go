package model

import "errors"

var (
	// ErrMissingPatient is returned when ComposePrescriptionCmd omits the patient
	// id.
	ErrMissingPatient = errors.New("prescription: patient id is required")

	// ErrMissingProvider is returned when ComposePrescriptionCmd omits the
	// provider id.
	ErrMissingProvider = errors.New("prescription: provider id is required")

	// ErrMissingMedication is returned when ComposePrescriptionCmd omits the
	// medication.
	ErrMissingMedication = errors.New("prescription: medication is required")

	// ErrMissingDosage is returned when ComposePrescriptionCmd omits the dosage.
	ErrMissingDosage = errors.New("prescription: dosage is required")

	// ErrProviderNotAuthorized is returned when the issuing provider is not an
	// authenticated provider with an active care relationship to the patient.
	// Invariant: a prescription may only be issued by an authenticated provider
	// with an active care relationship.
	ErrProviderNotAuthorized = errors.New("prescription: a prescription may only be issued by an authenticated provider with an active care relationship")

	// ErrSafetyCheckUnacknowledged is returned when the prescription failed an
	// allergy or interaction check that has not been acknowledged or overridden.
	// Invariant: a prescription failing an allergy or interaction check cannot be
	// transmitted until acknowledged/overridden.
	ErrSafetyCheckUnacknowledged = errors.New("prescription: a prescription failing an allergy or interaction check cannot be transmitted until acknowledged/overridden")

	// ErrTransmittedImmutable is returned when composing would alter a
	// prescription that has already been transmitted. Invariant: a transmitted
	// prescription is immutable and can only be superseded by a cancellation.
	ErrTransmittedImmutable = errors.New("prescription: a transmitted prescription is immutable and can only be superseded by a cancellation")
)
