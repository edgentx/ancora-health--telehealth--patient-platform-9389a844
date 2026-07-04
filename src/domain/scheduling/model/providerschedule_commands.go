package model

// BlockTimeCmd requests that an interval on a provider's schedule be marked
// unavailable so no bookable slot may be offered over it.
//
// Blocking time carves an off-limits window out of a provider's schedule — a
// break, an administrative hold, time reserved away from patients. Because a
// block reshapes the provider's published availability, the scheduling
// invariants gate it: availability windows for the same provider must not
// overlap, a blocked interval cannot simultaneously be offered as a bookable
// slot, and published availability must fall within the provider's clinic
// operating hours. ProviderId identifies the provider whose schedule is being
// blocked, Interval the span of time being marked unavailable, and Reason why
// the block is placed; all three are mandatory.
type BlockTimeCmd struct {
	// ProviderId identifies the provider whose schedule the interval is blocked on.
	ProviderId string
	// Interval identifies the span of time on the schedule being marked unavailable.
	Interval string
	// Reason records why the interval is being blocked.
	Reason string
}
