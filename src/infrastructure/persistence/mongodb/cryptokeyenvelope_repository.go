package mongodb

import (
	"context"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/repository"
)

// CryptoKeyEnvelopeCollection is the MongoDB collection key envelopes persist to.
const CryptoKeyEnvelopeCollection = "crypto_key_envelopes"

// cryptoKeyEnvelopeDoc is the BSON persistence shape of a
// CryptoKeyEnvelopeAggregate. It stores only the envelope's lifecycle metadata —
// whether the wrapping master key is active, when the envelope expires, and
// whether it has been revoked. The aggregate holds no raw key bytes, so no
// plaintext key material is ever mapped into this document: data keys only ever
// leave the crypto package wrapped inside a CipherText envelope.
type cryptoKeyEnvelopeDoc struct {
	DocID string `bson:"_id"`
	Ver   int    `bson:"version"`

	AggregateVersion int       `bson:"aggregate_version"`
	MasterKeyActive  bool      `bson:"master_key_active"`
	ExpiresAt        time.Time `bson:"expires_at"`
	Revoked          bool      `bson:"revoked"`
}

func (d *cryptoKeyEnvelopeDoc) ID() string       { return d.DocID }
func (d *cryptoKeyEnvelopeDoc) Version() int     { return d.Ver }
func (d *cryptoKeyEnvelopeDoc) SetVersion(v int) { d.Ver = v }

// CryptoKeyEnvelopeRepository is the MongoDB adapter for the
// CryptoKeyEnvelopeRepository port.
type CryptoKeyEnvelopeRepository struct {
	base *BaseRepository
}

// NewCryptoKeyEnvelopeRepository builds a repository over a document store.
func NewCryptoKeyEnvelopeRepository(store DocumentStore) *CryptoKeyEnvelopeRepository {
	return &CryptoKeyEnvelopeRepository{base: NewBaseRepository(store, CryptoKeyEnvelopeCollection)}
}

// Save persists the envelope with optimistic-concurrency semantics.
func (r *CryptoKeyEnvelopeRepository) Save(ctx context.Context, a *model.CryptoKeyEnvelopeAggregate) error {
	return upsert(ctx, r.base, cryptoKeyEnvelopeToDoc(a), &cryptoKeyEnvelopeDoc{})
}

// FindByID loads the envelope with the given id.
func (r *CryptoKeyEnvelopeRepository) FindByID(ctx context.Context, id string) (*model.CryptoKeyEnvelopeAggregate, error) {
	doc := &cryptoKeyEnvelopeDoc{}
	if err := r.base.FindByID(ctx, id, doc); err != nil {
		return nil, err
	}
	return cryptoKeyEnvelopeFromDoc(doc), nil
}

// cryptoKeyEnvelopeToDoc maps the aggregate onto its persistence document.
func cryptoKeyEnvelopeToDoc(a *model.CryptoKeyEnvelopeAggregate) *cryptoKeyEnvelopeDoc {
	return &cryptoKeyEnvelopeDoc{
		DocID:            a.ID,
		AggregateVersion: a.GetVersion(),
		MasterKeyActive:  a.MasterKeyActive,
		ExpiresAt:        a.ExpiresAt,
		Revoked:          a.Revoked,
	}
}

// cryptoKeyEnvelopeFromDoc reconstructs the aggregate from its document.
func cryptoKeyEnvelopeFromDoc(d *cryptoKeyEnvelopeDoc) *model.CryptoKeyEnvelopeAggregate {
	a := &model.CryptoKeyEnvelopeAggregate{
		ID:              d.DocID,
		MasterKeyActive: d.MasterKeyActive,
		ExpiresAt:       d.ExpiresAt,
		Revoked:         d.Revoked,
	}
	a.Version = d.AggregateVersion
	return a
}

// Compile-time assertion that the adapter satisfies its domain port.
var _ repository.CryptoKeyEnvelopeRepository = (*CryptoKeyEnvelopeRepository)(nil)
