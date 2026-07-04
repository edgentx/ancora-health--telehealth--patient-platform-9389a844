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
