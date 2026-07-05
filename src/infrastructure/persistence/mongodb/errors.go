package mongodb

import (
	"errors"
	"fmt"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// ErrDocumentNotFound is returned by the store and base repository when a lookup
// or delete targets an identity that does not exist.
var ErrDocumentNotFound = errors.New("mongodb: document not found")

// ErrDuplicateKey is returned when inserting a document whose id already exists.
var ErrDuplicateKey = errors.New("mongodb: duplicate key")

// OptimisticConcurrencyError is returned when an Update targets a version that no
// longer matches the stored document — i.e. another writer advanced the version
// first. It wraps shared.ErrConcurrencyConflict so callers can match it with
// errors.Is(err, shared.ErrConcurrencyConflict) while still recovering the
// collection, id and expected version for logging or retry.
type OptimisticConcurrencyError struct {
	Collection      string
	ID              string
	ExpectedVersion int
}

// Error renders the conflict with the identity and version that failed to match.
func (e *OptimisticConcurrencyError) Error() string {
	return fmt.Sprintf(
		"mongodb: optimistic concurrency conflict on %s/%s: expected version %d",
		e.Collection, e.ID, e.ExpectedVersion,
	)
}

// Unwrap bridges the typed error to the shared sentinel so existing
// errors.Is(..., shared.ErrConcurrencyConflict) checks keep working.
func (e *OptimisticConcurrencyError) Unwrap() error {
	return shared.ErrConcurrencyConflict
}
