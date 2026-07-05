package mongodb

import (
	"context"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/repository"
)

// SessionCollection is the MongoDB collection durable session records persist to.
const SessionCollection = "sessions"

// sessionDoc is the BSON persistence shape of a SessionAggregate. It carries no
// PHI, so it is stored verbatim through the BaseRepository. ExpiresAt is the
// session's TTL boundary: a Redis-backed cache can front this collection for
// fast, auto-expiring lookups while MongoDB holds the durable record of record.
type sessionDoc struct {
	DocID string `bson:"_id"`
	Ver   int    `bson:"version"`

	AggregateVersion  int       `bson:"aggregate_version"`
	Authenticated     bool      `bson:"authenticated"`
	Revoked           bool      `bson:"revoked"`
	Issued            bool      `bson:"issued"`
	AccountID         string    `bson:"account_id"`
	Role              string    `bson:"role"`
	DeviceFingerprint string    `bson:"device_fingerprint"`
	ExpiresAt         time.Time `bson:"expires_at"`
}

func (d *sessionDoc) ID() string       { return d.DocID }
func (d *sessionDoc) Version() int     { return d.Ver }
func (d *sessionDoc) SetVersion(v int) { d.Ver = v }

// SessionRepository is the MongoDB adapter for the SessionRepository port. It
// persists the durable session record; the ExpiresAt field is the boundary a
// TTL index (or a Redis cache in front of it) uses to expire the session.
type SessionRepository struct {
	base *BaseRepository
}

// NewSessionRepository builds a repository over a document store.
func NewSessionRepository(store DocumentStore) *SessionRepository {
	return &SessionRepository{base: NewBaseRepository(store, SessionCollection)}
}

// Save persists the session with optimistic-concurrency semantics.
func (r *SessionRepository) Save(ctx context.Context, a *model.SessionAggregate) error {
	return upsert(ctx, r.base, sessionToDoc(a), &sessionDoc{})
}

// FindByID loads the session with the given id.
func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.SessionAggregate, error) {
	doc := &sessionDoc{}
	if err := r.base.FindByID(ctx, id, doc); err != nil {
		return nil, err
	}
	return sessionFromDoc(doc), nil
}

// sessionToDoc maps the aggregate onto its persistence document.
func sessionToDoc(a *model.SessionAggregate) *sessionDoc {
	return &sessionDoc{
		DocID:             a.ID,
		AggregateVersion:  a.GetVersion(),
		Authenticated:     a.Authenticated,
		Revoked:           a.Revoked,
		Issued:            a.Issued,
		AccountID:         a.AccountID,
		Role:              a.Role,
		DeviceFingerprint: a.DeviceFingerprint,
		ExpiresAt:         a.ExpiresAt,
	}
}

// sessionFromDoc reconstructs the aggregate from its persistence document.
func sessionFromDoc(d *sessionDoc) *model.SessionAggregate {
	a := &model.SessionAggregate{
		ID:                d.DocID,
		Authenticated:     d.Authenticated,
		Revoked:           d.Revoked,
		Issued:            d.Issued,
		AccountID:         d.AccountID,
		Role:              d.Role,
		DeviceFingerprint: d.DeviceFingerprint,
		ExpiresAt:         d.ExpiresAt,
	}
	a.Version = d.AggregateVersion
	return a
}

// Compile-time assertion that the adapter satisfies its domain port.
var _ repository.SessionRepository = (*SessionRepository)(nil)
