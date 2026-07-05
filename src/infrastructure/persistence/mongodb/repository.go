package mongodb

import "context"

// VersionedDocument is the contract a persistence document must satisfy to be
// managed by BaseRepository. The version drives optimistic-concurrency control:
// every successful Update increments it, and a stale version is rejected.
type VersionedDocument interface {
	// ID returns the document identity, stored as MongoDB's _id.
	ID() string
	// Version returns the current optimistic-concurrency version.
	Version() int
	// SetVersion updates the in-memory version; the repository calls this as it
	// inserts (version 1) and updates (version+1) so the caller's document stays
	// consistent with what was persisted.
	SetVersion(v int)
}

// BaseRepository is the reusable persistence foundation shared by every
// bounded-context repository. It provides Insert/FindByID/Update/Delete over a
// DocumentStore with an incrementing version field, translating a version-guard
// miss into a typed OptimisticConcurrencyError.
type BaseRepository struct {
	store          DocumentStore
	collectionName string
}

// NewBaseRepository builds a BaseRepository over a store. collectionName is used
// only for error context (it identifies which collection a conflict occurred in).
func NewBaseRepository(store DocumentStore, collectionName string) *BaseRepository {
	return &BaseRepository{store: store, collectionName: collectionName}
}

// Insert persists a new document, stamping it with version 1. Concurrency
// control begins from the first update.
func (r *BaseRepository) Insert(ctx context.Context, doc VersionedDocument) error {
	doc.SetVersion(1)
	return r.store.InsertOne(ctx, doc.ID(), doc)
}

// FindByID decodes the document with the given id into dest, returning
// ErrDocumentNotFound if it does not exist.
func (r *BaseRepository) FindByID(ctx context.Context, id string, dest VersionedDocument) error {
	return r.store.FindOne(ctx, id, dest)
}

// Update replaces the document only if the store still holds the version the
// caller last read. It advances the version by one and, on a guard miss (another
// writer won the race), returns an *OptimisticConcurrencyError referencing the
// version that failed to match.
func (r *BaseRepository) Update(ctx context.Context, doc VersionedDocument) error {
	expected := doc.Version()
	doc.SetVersion(expected + 1)

	matched, err := r.store.ReplaceVersioned(ctx, doc.ID(), expected, doc)
	if err != nil {
		return err
	}
	if !matched {
		// Restore the caller's version so a retry re-reads from a clean state.
		doc.SetVersion(expected)
		return &OptimisticConcurrencyError{
			Collection:      r.collectionName,
			ID:              doc.ID(),
			ExpectedVersion: expected,
		}
	}
	return nil
}

// Delete removes the document with the given id, returning ErrDocumentNotFound
// if nothing was deleted.
func (r *BaseRepository) Delete(ctx context.Context, id string) error {
	deleted, err := r.store.DeleteOne(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrDocumentNotFound
	}
	return nil
}
