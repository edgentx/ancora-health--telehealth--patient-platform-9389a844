package shared

import "errors"

var (
	// ErrUnknownCommand is returned by Aggregate.Execute when the supplied
	// command type is not recognized by the aggregate.
	ErrUnknownCommand = errors.New("unknown command")

	// ErrConcurrencyConflict signals an optimistic-concurrency version mismatch
	// when persisting an aggregate whose version no longer matches the store.
	ErrConcurrencyConflict = errors.New("concurrency conflict")
)
