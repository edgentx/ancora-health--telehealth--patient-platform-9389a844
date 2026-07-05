package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// upsert persists doc with the BaseRepository's optimistic-concurrency
// semantics behind the single Save entrypoint every domain port exposes: it
// inserts when the identity is absent (stamping version 1) and performs a
// version-guarded replace otherwise. current must be a fresh, zero-valued
// document of doc's concrete type; it is used only to read the currently stored
// version that the replace guards against. A guard miss surfaces, unchanged, as
// the BaseRepository's *OptimisticConcurrencyError.
func upsert(ctx context.Context, base *BaseRepository, doc, current VersionedDocument) error {
	err := base.FindByID(ctx, doc.ID(), current)
	switch {
	case errors.Is(err, ErrDocumentNotFound):
		return base.Insert(ctx, doc)
	case err != nil:
		return err
	}
	doc.SetVersion(current.Version())
	return base.Update(ctx, doc)
}

// encryptedStore persists documents whose PHI/PII fields must be encrypted at
// rest. It mirrors the BaseRepository's insert / version-guarded-replace flow
// but routes every write through the S-68 crypto codec, so a stored document
// never contains PHI plaintext, and every read back through the codec so the
// caller receives the decrypted domain values. Timestamps in an encrypted
// document must be stored as epoch millis (see epochMillis): the codec
// round-trips through bson.M, and BSON decodes datetimes as an int64-backed
// primitive the reflective codec cannot re-assign to a time.Time field.
type encryptedStore struct {
	store      DocumentStore
	codec      *crypto.Codec
	collection string
}

// newEncryptedStore builds an encryptedStore over a document store and codec.
// collection is used only for error context on a concurrency conflict.
func newEncryptedStore(store DocumentStore, codec *crypto.Codec, collection string) *encryptedStore {
	return &encryptedStore{store: store, codec: codec, collection: collection}
}

// save encrypts doc and persists it, inserting when the identity is absent
// (stamped version 1) or version-guarded replacing it otherwise. A guard miss
// yields an *OptimisticConcurrencyError, exactly as the BaseRepository would.
func (e *encryptedStore) save(ctx context.Context, doc VersionedDocument) error {
	var raw bson.M
	err := e.store.FindOne(ctx, doc.ID(), &raw)
	switch {
	case errors.Is(err, ErrDocumentNotFound):
		doc.SetVersion(1)
		enc, err := e.codec.EncryptDocument(ctx, doc)
		if err != nil {
			return err
		}
		return e.store.InsertOne(ctx, doc.ID(), enc)
	case err != nil:
		return err
	}

	expected := coerceVersion(raw[versionField])
	doc.SetVersion(expected + 1)
	enc, err := e.codec.EncryptDocument(ctx, doc)
	if err != nil {
		return err
	}
	matched, err := e.store.ReplaceVersioned(ctx, doc.ID(), expected, enc)
	if err != nil {
		return err
	}
	if !matched {
		doc.SetVersion(expected)
		return &OptimisticConcurrencyError{
			Collection:      e.collection,
			ID:              doc.ID(),
			ExpectedVersion: expected,
		}
	}
	return nil
}

// load reads the document with id, decrypting its PHI/PII fields into dest.
func (e *encryptedStore) load(ctx context.Context, id string, dest VersionedDocument) error {
	var raw bson.M
	if err := e.store.FindOne(ctx, id, &raw); err != nil {
		return err
	}
	return e.codec.DecryptDocument(ctx, raw, dest)
}

// coerceVersion reads a stored version field back into an int, tolerating the
// int32/int64 forms BSON decodes numbers into.
func coerceVersion(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

// epochMillis encodes a timestamp as UTC milliseconds since the Unix epoch for
// storage in an encrypted document, mapping the zero time to 0. Storing times as
// an integer keeps them round-trippable through the codec's bson.M path.
func epochMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UTC().UnixMilli()
}

// fromEpochMillis reverses epochMillis, mapping 0 back to the zero time.
func fromEpochMillis(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}
