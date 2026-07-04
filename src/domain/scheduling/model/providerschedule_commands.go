package model

// PublishAvailabilityCmd requests that a provider publish a set of bookable
// availability windows onto their schedule.
//
// Publishing availability is the act that opens a provider's calendar for
// booking: it declares the windows patients may hold slots within. The
// scheduling invariants gate the publish — availability windows for the same
// provider must not overlap, an interval the provider has blocked cannot be
// offered as a bookable slot, and published availability must fall within the
// provider's clinic operating hours. ProviderId identifies the provider whose
// schedule is being published and Windows carries the availability windows being
// offered; both are mandatory.
type PublishAvailabilityCmd struct {
	// ProviderId identifies the provider whose availability is being published.
	ProviderId string
	// Windows carries the bookable availability windows being offered on the
	// provider's schedule. At least one window must be present.
	Windows []string
}
