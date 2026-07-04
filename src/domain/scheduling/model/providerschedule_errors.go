package model

import "errors"

var (
	// ErrMissingProviderScheduleProvider is returned when BlockTimeCmd omits the
	// provider id.
	ErrMissingProviderScheduleProvider = errors.New("providerschedule: provider id is required")

	// ErrMissingBlockInterval is returned when BlockTimeCmd omits the interval to
	// block.
	ErrMissingBlockInterval = errors.New("providerschedule: interval is required")

	// ErrMissingBlockReason is returned when BlockTimeCmd omits the reason for the
	// block.
	ErrMissingBlockReason = errors.New("providerschedule: reason is required")

	// ErrAvailabilityWindowsOverlap is returned when the block would leave
	// overlapping availability windows for the provider. Invariant: availability
	// windows for the same provider must not overlap.
	ErrAvailabilityWindowsOverlap = errors.New("providerschedule: availability windows for the same provider must not overlap")

	// ErrBlockedIntervalBookable is returned when the interval being blocked is
	// still offered as a bookable slot. Invariant: a blocked interval cannot be
	// offered as a bookable slot.
	ErrBlockedIntervalBookable = errors.New("providerschedule: a blocked interval cannot be offered as a bookable slot")

	// ErrAvailabilityOutsideOperatingHours is returned when published availability
	// falls outside the provider's clinic operating hours. Invariant: published
	// availability must fall within the provider's clinic operating hours.
	ErrAvailabilityOutsideOperatingHours = errors.New("providerschedule: published availability must fall within the provider's clinic operating hours")
)
