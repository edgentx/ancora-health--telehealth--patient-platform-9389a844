package mongodb

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
)

// MemStore is an in-memory DocumentStore for local development and tests. It
// serializes documents through BSON (exactly as the MongoDB adapter would), so
// it exercises the same marshaling path and enforces the same version semantics
// — including the atomic compare-and-swap that ReplaceVersioned relies on for
// optimistic concurrency. It is safe for concurrent use.
type MemStore struct {
	mu   sync.Mutex
	docs map[string][]byte
}

// NewMemStore builds an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{docs: make(map[string][]byte)}
}

// InsertOne stores a new document, rejecting a duplicate id.
func (m *MemStore) InsertOne(_ context.Context, id string, doc any) error {
	data, err := bson.Marshal(doc)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.docs[id]; exists {
		return ErrDuplicateKey
	}
	m.docs[id] = data
	return nil
}

// FindOne decodes the stored document into dest.
func (m *MemStore) FindOne(_ context.Context, id string, dest any) error {
	m.mu.Lock()
	data, ok := m.docs[id]
	m.mu.Unlock()
	if !ok {
		return ErrDocumentNotFound
	}
	return bson.Unmarshal(data, dest)
}

// ReplaceVersioned performs the version-guarded compare-and-swap under the lock,
// so two concurrent updates from the same stale version cannot both succeed.
func (m *MemStore) ReplaceVersioned(_ context.Context, id string, expectedVersion int, doc any) (bool, error) {
	data, err := bson.Marshal(doc)
	if err != nil {
		return false, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	stored, ok := m.docs[id]
	if !ok {
		return false, nil
	}
	if storedVersion(stored) != expectedVersion {
		return false, nil
	}
	m.docs[id] = data
	return true, nil
}

// DeleteOne removes a document, reporting whether it existed.
func (m *MemStore) DeleteOne(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.docs[id]; !ok {
		return false, nil
	}
	delete(m.docs, id)
	return true, nil
}

// RunInTransaction gives MemStore the TransactionRunner behavior: it snapshots
// the store, runs the unit of work, and restores the snapshot if the work
// returns an error — an in-memory analogue of a MongoDB transaction rollback.
func (m *MemStore) RunInTransaction(ctx context.Context, work UnitOfWork) error {
	snapshot := m.snapshot()
	if err := work(ctx); err != nil {
		m.restore(snapshot)
		return err
	}
	return nil
}

// snapshot copies the current document set so it can be restored on rollback.
func (m *MemStore) snapshot() map[string][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make(map[string][]byte, len(m.docs))
	for k, v := range m.docs {
		buf := make([]byte, len(v))
		copy(buf, v)
		copied[k] = buf
	}
	return copied
}

// restore replaces the document set with a previously captured snapshot.
func (m *MemStore) restore(snapshot map[string][]byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.docs = snapshot
}

// storedVersion reads the version field out of a marshaled document.
func storedVersion(data []byte) int {
	var env struct {
		Version int `bson:"version"`
	}
	if err := bson.Unmarshal(data, &env); err != nil {
		return -1
	}
	return env.Version
}
