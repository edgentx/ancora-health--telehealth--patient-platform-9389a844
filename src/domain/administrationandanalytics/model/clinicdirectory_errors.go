package model

import "errors"

var (
	// ErrMissingProvider is returned when RegisterProviderCmd omits the provider
	// id.
	ErrMissingProvider = errors.New("clinicdirectory: provider id is required")

	// ErrMissingSpecialties is returned when RegisterProviderCmd supplies no
	// specialty codes.
	ErrMissingSpecialties = errors.New("clinicdirectory: at least one specialty is required")

	// ErrMissingClinics is returned when RegisterProviderCmd supplies no clinic
	// ids.
	ErrMissingClinics = errors.New("clinicdirectory: at least one clinic is required")

	// ErrMissingSpecialtyCode is returned when ManageSpecialtyCmd omits the
	// specialty code.
	ErrMissingSpecialtyCode = errors.New("clinicdirectory: specialty code is required")

	// ErrMissingDisplayName is returned when ManageSpecialtyCmd omits the display
	// name.
	ErrMissingDisplayName = errors.New("clinicdirectory: specialty display name is required")

	// ErrProviderNotBookable is returned when registering a provider that is not
	// assigned to any clinic. Invariant: a provider must be assigned to at least
	// one clinic to be bookable.
	ErrProviderNotBookable = errors.New("clinicdirectory: a provider must be assigned to at least one clinic to be bookable")

	// ErrDuplicateSpecialtyCode is returned when registering a provider would
	// introduce a specialty code that already exists in the directory. Invariant:
	// a specialty code must be unique within the directory.
	ErrDuplicateSpecialtyCode = errors.New("clinicdirectory: a specialty code must be unique within the directory")

	// ErrClinicDeactivationBlocked is returned when a clinic with future booked
	// appointments would be deactivated. Invariant: a clinic cannot be deactivated
	// while it has future booked appointments.
	ErrClinicDeactivationBlocked = errors.New("clinicdirectory: a clinic cannot be deactivated while it has future booked appointments")
)
