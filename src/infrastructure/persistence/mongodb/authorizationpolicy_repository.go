package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// AuthorizationPolicyCollection is the MongoDB collection policies persist to.
const AuthorizationPolicyCollection = "authorization_policies"

// authorizationPolicyDoc is the BSON persistence shape of an
// AuthorizationPolicyAggregate. The publishing author is PII and is tagged
// `phi:"true"` so the codec encrypts it at rest; the Rego bundle and the
// invariant flags are policy configuration and persist in the clear.
type authorizationPolicyDoc struct {
	DocID string `bson:"_id"`
	Ver   int    `bson:"version"`

	AggregateVersion int    `bson:"aggregate_version"`
	Status           string `bson:"status"`
	RegoBundle       string `bson:"rego_bundle"`
	Author           string `bson:"author" phi:"true"`

	DefaultDenyMissing     bool `bson:"default_deny_missing"`
	PermissionAboveCeiling bool `bson:"permission_above_ceiling"`
	NonBinaryDecision      bool `bson:"non_binary_decision"`
	PHIScopingMissing      bool `bson:"phi_scoping_missing"`
	AccessAllowed          bool `bson:"access_allowed"`

	LastDecision string `bson:"last_decision"`
}

func (d *authorizationPolicyDoc) ID() string       { return d.DocID }
func (d *authorizationPolicyDoc) Version() int     { return d.Ver }
func (d *authorizationPolicyDoc) SetVersion(v int) { d.Ver = v }

// AuthorizationPolicyRepository is the MongoDB adapter for the
// AuthorizationPolicyRepository port. The policy author (PII) is encrypted
// through the S-68 crypto codec before it reaches storage.
type AuthorizationPolicyRepository struct {
	store *encryptedStore
}

// NewAuthorizationPolicyRepository builds a repository over a store and codec.
func NewAuthorizationPolicyRepository(store DocumentStore, codec *crypto.Codec) *AuthorizationPolicyRepository {
	return &AuthorizationPolicyRepository{store: newEncryptedStore(store, codec, AuthorizationPolicyCollection)}
}

// Save persists the policy, encrypting its PII before it reaches the store.
func (r *AuthorizationPolicyRepository) Save(ctx context.Context, a *model.AuthorizationPolicyAggregate) error {
	return r.store.save(ctx, authorizationPolicyToDoc(a))
}

// FindByID loads and decrypts the policy with the given id.
func (r *AuthorizationPolicyRepository) FindByID(ctx context.Context, id string) (*model.AuthorizationPolicyAggregate, error) {
	doc := &authorizationPolicyDoc{}
	if err := r.store.load(ctx, id, doc); err != nil {
		return nil, err
	}
	return authorizationPolicyFromDoc(doc), nil
}

// authorizationPolicyToDoc maps the aggregate onto its persistence document.
func authorizationPolicyToDoc(a *model.AuthorizationPolicyAggregate) *authorizationPolicyDoc {
	return &authorizationPolicyDoc{
		DocID:                  a.ID,
		AggregateVersion:       a.GetVersion(),
		Status:                 string(a.Status),
		RegoBundle:             a.RegoBundle,
		Author:                 a.Author,
		DefaultDenyMissing:     a.DefaultDenyMissing,
		PermissionAboveCeiling: a.PermissionAboveCeiling,
		NonBinaryDecision:      a.NonBinaryDecision,
		PHIScopingMissing:      a.PHIScopingMissing,
		AccessAllowed:          a.AccessAllowed,
		LastDecision:           a.LastDecision,
	}
}

// authorizationPolicyFromDoc reconstructs the aggregate from its document.
func authorizationPolicyFromDoc(d *authorizationPolicyDoc) *model.AuthorizationPolicyAggregate {
	a := &model.AuthorizationPolicyAggregate{
		ID:                     d.DocID,
		Status:                 model.PolicyStatus(d.Status),
		RegoBundle:             d.RegoBundle,
		Author:                 d.Author,
		DefaultDenyMissing:     d.DefaultDenyMissing,
		PermissionAboveCeiling: d.PermissionAboveCeiling,
		NonBinaryDecision:      d.NonBinaryDecision,
		PHIScopingMissing:      d.PHIScopingMissing,
		AccessAllowed:          d.AccessAllowed,
		LastDecision:           d.LastDecision,
	}
	a.Version = d.AggregateVersion
	return a
}

// Compile-time assertion that the adapter satisfies its domain port.
var _ repository.AuthorizationPolicyRepository = (*AuthorizationPolicyRepository)(nil)
