package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// CareRelationshipCollection is the MongoDB collection care relationships
// persist to.
const CareRelationshipCollection = "care_relationships"

// careRelationshipDoc is the BSON persistence shape of a
// CareRelationshipAggregate. The fact that a specific provider treats a specific
// patient is PHI, so the provider, patient, and scoped-role account identifiers
// are tagged `phi:"true"` and encrypted at rest; the clinic scoping and the
// invariant flags persist in the clear.
type careRelationshipDoc struct {
	DocID string `bson:"_id"`
	Ver   int    `bson:"version"`

	AggregateVersion int    `bson:"aggregate_version"`
	Status           string `bson:"status"`
	ProviderID       string `bson:"provider_id" phi:"true"`
	PatientID        string `bson:"patient_id" phi:"true"`
	ClinicID         string `bson:"clinic_id"`

	Inactive     bool `bson:"inactive"`
	EpisodeEnded bool `bson:"episode_ended"`
	SelfAsserted bool `bson:"self_asserted"`

	ScopedRoleAccountID string `bson:"scoped_role_account_id" phi:"true"`
	ScopedRole          string `bson:"scoped_role"`
	ScopedRoleClinicID  string `bson:"scoped_role_clinic_id"`
}

func (d *careRelationshipDoc) ID() string       { return d.DocID }
func (d *careRelationshipDoc) Version() int     { return d.Ver }
func (d *careRelationshipDoc) SetVersion(v int) { d.Ver = v }

// CareRelationshipRepository is the MongoDB adapter for the
// CareRelationshipRepository port. Provider/patient identifiers (PHI) are
// encrypted through the S-68 crypto codec before they reach storage.
type CareRelationshipRepository struct {
	store *encryptedStore
}

// NewCareRelationshipRepository builds a repository over a store and codec.
func NewCareRelationshipRepository(store DocumentStore, codec *crypto.Codec) *CareRelationshipRepository {
	return &CareRelationshipRepository{store: newEncryptedStore(store, codec, CareRelationshipCollection)}
}

// Save persists the relationship, encrypting its PHI before it reaches the store.
func (r *CareRelationshipRepository) Save(ctx context.Context, a *model.CareRelationshipAggregate) error {
	return r.store.save(ctx, careRelationshipToDoc(a))
}

// FindByID loads and decrypts the relationship with the given id.
func (r *CareRelationshipRepository) FindByID(ctx context.Context, id string) (*model.CareRelationshipAggregate, error) {
	doc := &careRelationshipDoc{}
	if err := r.store.load(ctx, id, doc); err != nil {
		return nil, err
	}
	return careRelationshipFromDoc(doc), nil
}

// careRelationshipToDoc maps the aggregate onto its persistence document.
func careRelationshipToDoc(a *model.CareRelationshipAggregate) *careRelationshipDoc {
	return &careRelationshipDoc{
		DocID:               a.ID,
		AggregateVersion:    a.GetVersion(),
		Status:              string(a.Status),
		ProviderID:          a.ProviderID,
		PatientID:           a.PatientID,
		ClinicID:            a.ClinicID,
		Inactive:            a.Inactive,
		EpisodeEnded:        a.EpisodeEnded,
		SelfAsserted:        a.SelfAsserted,
		ScopedRoleAccountID: a.ScopedRoleAccountID,
		ScopedRole:          a.ScopedRole,
		ScopedRoleClinicID:  a.ScopedRoleClinicID,
	}
}

// careRelationshipFromDoc reconstructs the aggregate from its document.
func careRelationshipFromDoc(d *careRelationshipDoc) *model.CareRelationshipAggregate {
	a := &model.CareRelationshipAggregate{
		ID:                  d.DocID,
		Status:              model.RelationshipStatus(d.Status),
		ProviderID:          d.ProviderID,
		PatientID:           d.PatientID,
		ClinicID:            d.ClinicID,
		Inactive:            d.Inactive,
		EpisodeEnded:        d.EpisodeEnded,
		SelfAsserted:        d.SelfAsserted,
		ScopedRoleAccountID: d.ScopedRoleAccountID,
		ScopedRole:          d.ScopedRole,
		ScopedRoleClinicID:  d.ScopedRoleClinicID,
	}
	a.Version = d.AggregateVersion
	return a
}

// Compile-time assertion that the adapter satisfies its domain port.
var _ repository.CareRelationshipRepository = (*CareRelationshipRepository)(nil)
