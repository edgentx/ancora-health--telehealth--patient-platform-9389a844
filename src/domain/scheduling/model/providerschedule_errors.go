package model

import "errors"

var (
	// ErrMissingScheduleProvider is returned when PublishAvailabilityCmd omits the
	// provider id.
	ErrMissingScheduleProvider = errors.New("providerschedule: provider id is required")

	// ErrMissingWindows is returned when PublishAvailabilityCmd omits the
	// availability windows.
	ErrMissingWindows = errors.New("providerschedule: at least one availability window is required")

	// ErrOverlappingWindows is returned when the published windows overlap one
	// another. Invariant: availability windows for the same provider must not
	// overlap.
	ErrOverlappingWindows = errors.New("providerschedule: availability windows for the same provider must not overlap")

	// ErrBlockedIntervalOffered is returned when a published window offers an
	// interval the provider has blocked. Invariant: a blocked interval cannot be
	// offered as a bookable slot.
	ErrBlockedIntervalOffered = errors.New("providerschedule: a blocked interval cannot be offered as a bookable slot")

	// ErrOutsideOperatingHours is returned when published availability falls
	// outside the provider's clinic operating hours. Invariant: published
	// availability must fall within the provider's clinic operating hours.
	ErrOutsideOperatingHours = errors.New("providerschedule: published availability must fall within the provider's clinic operating hours")
)
