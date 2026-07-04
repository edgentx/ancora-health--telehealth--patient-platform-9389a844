package model

// RegisterProviderCmd requests that a provider be added to the clinic
// directory, capturing the provider being registered, the specialties they
// practice, and the clinics they are assigned to.
//
// Registering a provider is the act that makes them discoverable and bookable
// within the directory: a provider must be assigned to at least one clinic to be
// bookable, a specialty code must be unique within the directory, and a clinic
// cannot be deactivated while it has future booked appointments. ProviderId
// identifies the provider, Specialties the specialty codes they practice, and
// ClinicIds the clinics they are assigned to. All three are mandatory, and at
// least one specialty and one clinic must be supplied.
type RegisterProviderCmd struct {
	// ProviderId identifies the provider being registered into the directory.
	ProviderId string
	// Specialties are the specialty codes the provider practices. At least one is
	// required.
	Specialties []string
	// ClinicIds are the clinics the provider is assigned to. At least one is
	// required.
	ClinicIds []string
}

// ManageSpecialtyCmd creates or updates a specialty entry in the clinic
// directory, capturing the specialty code that identifies the entry and the
// display name shown for it.
//
// Managing a specialty is an upsert: a code that is not yet in the directory is
// created, and one that already exists has its display name updated. The same
// directory invariants apply as for provider registration: a provider must be
// assigned to at least one clinic to be bookable, a specialty code must be
// unique within the directory, and a clinic cannot be deactivated while it has
// future booked appointments. Both SpecialtyCode and DisplayName are mandatory.
type ManageSpecialtyCmd struct {
	// SpecialtyCode identifies the specialty entry being created or updated.
	SpecialtyCode string
	// DisplayName is the human-readable name shown for the specialty.
	DisplayName string
}

// ConfigureClinicCmd creates or updates a clinic entry in the clinic directory,
// capturing the clinic identity that names the entry and the operating hours it
// is available for booking.
//
// Configuring a clinic is an upsert: an identity that is not yet in the
// directory is created, and one that already exists has its operating hours
// updated. The same directory invariants apply as for the other directory
// commands: a provider must be assigned to at least one clinic to be bookable, a
// specialty code must be unique within the directory, and a clinic cannot be
// deactivated while it has future booked appointments. Both ClinicIdentity and
// OperatingHours are mandatory.
type ConfigureClinicCmd struct {
	// ClinicIdentity identifies the clinic entry being created or updated.
	ClinicIdentity string
	// OperatingHours are the hours the clinic is available for booking.
	OperatingHours string
}
