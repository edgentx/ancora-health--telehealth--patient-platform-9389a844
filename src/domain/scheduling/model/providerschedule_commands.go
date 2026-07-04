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

// BlockTimeCmd requests that an interval on a provider's schedule be marked
// unavailable so that no bookable slot may be offered within it.
//
// Blocking time is the inverse of publishing availability: it carves an interval
// out of the provider's calendar for leave, an appointment held outside the
// booking system, or any other reason the interval cannot be offered. The same
// scheduling invariants gate the block — availability windows for the same
// provider must not overlap, a blocked interval cannot be offered as a bookable
// slot, and published availability must fall within the provider's clinic
// operating hours. ProviderId identifies the provider whose schedule is being
// blocked, Interval carries the span being marked unavailable, and Reason records
// why; all three are mandatory.
type BlockTimeCmd struct {
	// ProviderId identifies the provider whose time is being blocked.
	ProviderId string
	// Interval is the span on the provider's schedule being marked unavailable.
	Interval string
	// Reason records why the interval is being blocked.
	Reason string
}
