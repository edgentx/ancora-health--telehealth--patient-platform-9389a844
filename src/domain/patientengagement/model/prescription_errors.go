package model

import "errors"

var (
	// ErrMissingPrescriptionID is returned when RunSafetyCheckCmd omits the
	// prescription id.
	ErrMissingPrescriptionID = errors.New("prescription: prescription id is required")

	// ErrMissingPrescriptionProvider is returned when RunSafetyCheckCmd omits the
	// provider id.
	ErrMissingPrescriptionProvider = errors.New("prescription: provider id is required")

	// ErrMissingPrescriptionPatient is returned when RunSafetyCheckCmd omits the
	// patient id.
	ErrMissingPrescriptionPatient = errors.New("prescription: patient id is required")

	// ErrProviderNotInCare is returned when the issuing provider is not
	// authenticated or does not hold an active care relationship to the patient.
	// Invariant: a prescription may only be issued by an authenticated provider
	// with an active care relationship.
	ErrProviderNotInCare = errors.New("prescription: a prescription may only be issued by an authenticated provider with an active care relationship")

	// ErrUnacknowledgedSafetyFailure is returned when the prescription carries an
	// allergy or interaction failure that has not been acknowledged or overridden.
	// Invariant: a prescription failing an allergy or interaction check cannot be
	// transmitted until acknowledged/overridden.
	ErrUnacknowledgedSafetyFailure = errors.New("prescription: a prescription failing an allergy or interaction check cannot be transmitted until acknowledged/overridden")

	// ErrPrescriptionTransmitted is returned when acting on an already-transmitted
	// prescription. Invariant: a transmitted prescription is immutable and can only
	// be superseded by a cancellation.
	ErrPrescriptionTransmitted = errors.New("prescription: a transmitted prescription is immutable and can only be superseded by a cancellation")
)
