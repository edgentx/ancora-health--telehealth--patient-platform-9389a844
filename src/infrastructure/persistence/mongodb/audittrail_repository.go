package mongodb

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/repository"
)

// AuditTrailCollection is the MongoDB collection sealed audit entries persist to.
const AuditTrailCollection = "audit_trail_entries"

// auditEntryDoc is the immutable per-entry document written to the append-only
// audit collection. Its _id is derived from the trail id and the entry's
// sequence, so re-sealing an existing entry collides on insert and is rejected:
// the collection only ever grows, which is what makes the trail tamper-evident
// at the storage layer in addition to the in-chain hash.
type auditEntryDoc struct {
	DocID        string    `bson:"_id"`
	TrailID      string    `bson:"trail_id"`
	Sequence     int       `bson:"sequence"`
	ActorContext string    `bson:"actor_context"`
	ResourceRef  string    `bson:"resource_ref"`
	Action       string    `bson:"action"`
	OccurredAt   time.Time `bson:"occurred_at"`
	PrevHash     string    `bson:"prev_hash"`
	Hash         string    `bson:"hash"`
}

// auditEntryID derives the deterministic _id of an entry from its trail and
// sequence number.
func auditEntryID(trailID string, seq int) string {
	return trailID + "#" + strconv.Itoa(seq)
}

// AuditEntryCollection is the append-only persistence port for sealed audit
// entries. It deliberately exposes only Append and Load — no update or delete —
// so the immutability of the trail is enforced by the shape of the interface,
// not merely by convention. Append reports ErrDuplicateKey when an entry with
// the same identity already exists.
type AuditEntryCollection interface {
	Append(ctx context.Context, doc auditEntryDoc) error
	Load(ctx context.Context, trailID string) ([]auditEntryDoc, error)
}

// mongoAuditEntryCollection is the MongoDB-backed AuditEntryCollection.
type mongoAuditEntryCollection struct {
	collection *mongo.Collection
}

// NewMongoAuditEntryCollection builds an append-only entry collection over a
// MongoDB collection.
func NewMongoAuditEntryCollection(collection *mongo.Collection) AuditEntryCollection {
	return &mongoAuditEntryCollection{collection: collection}
}

func (c *mongoAuditEntryCollection) Append(ctx context.Context, doc auditEntryDoc) error {
	_, err := c.collection.InsertOne(ctx, doc)
	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicateKey
	}
	return err
}

func (c *mongoAuditEntryCollection) Load(ctx context.Context, trailID string) ([]auditEntryDoc, error) {
	cursor, err := c.collection.Find(
		ctx,
		bson.M{"trail_id": trailID},
		options.Find().SetSort(bson.D{{Key: "sequence", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	var docs []auditEntryDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// memAuditEntryCollection is an in-memory AuditEntryCollection for hermetic
// tests and local development. Like the MongoDB adapter it rejects a duplicate
// entry id, so append-only semantics are exercised without a live database.
type memAuditEntryCollection struct {
	mu      sync.Mutex
	byID    map[string]auditEntryDoc
	byTrail map[string][]auditEntryDoc
}

// NewMemAuditEntryCollection builds an empty in-memory entry collection.
func NewMemAuditEntryCollection() AuditEntryCollection {
	return &memAuditEntryCollection{
		byID:    make(map[string]auditEntryDoc),
		byTrail: make(map[string][]auditEntryDoc),
	}
}

func (c *memAuditEntryCollection) Append(_ context.Context, doc auditEntryDoc) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.byID[doc.DocID]; exists {
		return ErrDuplicateKey
	}
	c.byID[doc.DocID] = doc
	c.byTrail[doc.TrailID] = append(c.byTrail[doc.TrailID], doc)
	return nil
}

func (c *memAuditEntryCollection) Load(_ context.Context, trailID string) ([]auditEntryDoc, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	docs := make([]auditEntryDoc, len(c.byTrail[trailID]))
	copy(docs, c.byTrail[trailID])
	sort.Slice(docs, func(i, j int) bool { return docs[i].Sequence < docs[j].Sequence })
	return docs, nil
}

// AuditTrailRepository is the MongoDB adapter for the AuditTrailRepository port.
// It persists each sealed entry as its own immutable document in an append-only
// collection, so updates and deletes of an existing entry are structurally
// impossible; tampering that survives long enough to be re-saved is caught by
// the hash comparison in Save and, on read-back, by VerifyChainIntegrityCmd.
type AuditTrailRepository struct {
	entries AuditEntryCollection
}

// NewAuditTrailRepository builds a repository over an append-only entry
// collection.
func NewAuditTrailRepository(entries AuditEntryCollection) *AuditTrailRepository {
	return &AuditTrailRepository{entries: entries}
}

// NewMongoAuditTrailRepository builds a repository over a MongoDB collection.
func NewMongoAuditTrailRepository(collection *mongo.Collection) *AuditTrailRepository {
	return NewAuditTrailRepository(NewMongoAuditEntryCollection(collection))
}

// Save appends the trail's not-yet-persisted entries in chain order. An entry
// that is already sealed is immutable: an identical re-save is a no-op, while a
// re-save whose content differs from what is stored (a tamper attempt) is
// rejected with ErrAuditEntryImmutable rather than overwriting history.
func (r *AuditTrailRepository) Save(ctx context.Context, a *model.AuditTrailAggregate) error {
	existing, err := r.entries.Load(ctx, a.ID)
	if err != nil {
		return err
	}
	storedBySeq := make(map[int]auditEntryDoc, len(existing))
	for _, d := range existing {
		storedBySeq[d.Sequence] = d
	}

	for _, entry := range a.Entries() {
		if prior, ok := storedBySeq[entry.Sequence]; ok {
			if prior.Hash != entry.Hash {
				return model.ErrAuditEntryImmutable
			}
			continue
		}
		if err := r.entries.Append(ctx, auditEntryToDoc(a.ID, entry)); err != nil {
			if errors.Is(err, ErrDuplicateKey) {
				return model.ErrAuditEntryImmutable
			}
			return err
		}
	}
	return nil
}

// FindByID rehydrates the trail from its persisted entries, returning
// ErrDocumentNotFound when no entries exist for the id.
func (r *AuditTrailRepository) FindByID(ctx context.Context, id string) (*model.AuditTrailAggregate, error) {
	docs, err := r.entries.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, ErrDocumentNotFound
	}
	entries := make([]model.AuditEntry, len(docs))
	for i, d := range docs {
		entries[i] = auditEntryFromDoc(d)
	}
	return model.RehydrateAuditTrail(id, entries), nil
}

// auditEntryToDoc maps a sealed entry onto its immutable persistence document.
func auditEntryToDoc(trailID string, e model.AuditEntry) auditEntryDoc {
	return auditEntryDoc{
		DocID:        auditEntryID(trailID, e.Sequence),
		TrailID:      trailID,
		Sequence:     e.Sequence,
		ActorContext: e.ActorContext,
		ResourceRef:  e.ResourceRef,
		Action:       e.Action,
		OccurredAt:   e.OccurredAt,
		PrevHash:     e.PrevHash,
		Hash:         e.Hash,
	}
}

// auditEntryFromDoc reconstructs a sealed entry from its persistence document.
func auditEntryFromDoc(d auditEntryDoc) model.AuditEntry {
	return model.AuditEntry{
		Sequence:     d.Sequence,
		ActorContext: d.ActorContext,
		ResourceRef:  d.ResourceRef,
		Action:       d.Action,
		OccurredAt:   d.OccurredAt,
		PrevHash:     d.PrevHash,
		Hash:         d.Hash,
	}
}

// Compile-time assertion that the adapter satisfies its domain port.
var _ repository.AuditTrailRepository = (*AuditTrailRepository)(nil)
