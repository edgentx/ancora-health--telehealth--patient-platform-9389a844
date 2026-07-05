package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// insurancePoliciesCollection is the collection insurance-policy documents live in.
const insurancePoliciesCollection = "insurance_policies"

// insurancePolicyDoc is the at-rest projection of an InsurancePolicyAggregate.
// The payer identifier is PHI (it ties a patient to their coverage) and is stored
// encrypted.
type insurancePolicyDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	PatientID             string            `bson:"patient_id"`
	PayerIdentifierCipher crypto.CipherText `bson:"payer_identifier_cipher"`
	EffectiveStart        string            `bson:"effective_start"`
	EffectiveEnd          string            `bson:"effective_end"`
	VerifiedServiceDate   string            `bson:"verified_service_date"`

	EligibilityNotVerified    bool `bson:"eligibility_not_verified"`
	ActivePrimaryPolicyExists bool `bson:"active_primary_policy_exists"`
	PolicyExpired             bool `bson:"policy_expired"`
}

func (d *insurancePolicyDoc) ID() string       { return d.DocID }
func (d *insurancePolicyDoc) Version() int     { return d.Ver }
func (d *insurancePolicyDoc) SetVersion(v int) { d.Ver = v }

// InsurancePolicyRepository is the MongoDB-backed InsurancePolicyRepository,
// encrypting the payer identifier with the S-68 envelope cipher.
type InsurancePolicyRepository struct {
	base   *BaseRepository
	cipher *crypto.FieldCipher
}

var _ billrepo.InsurancePolicyRepository = (*InsurancePolicyRepository)(nil)

// NewInsurancePolicyRepository builds an insurance-policy repository over a store and cipher.
func NewInsurancePolicyRepository(store DocumentStore, cipher *crypto.FieldCipher) *InsurancePolicyRepository {
	return &InsurancePolicyRepository{base: NewBaseRepository(store, insurancePoliciesCollection), cipher: cipher}
}

// Save encrypts the payer identifier and persists the policy with optimistic concurrency.
func (r *InsurancePolicyRepository) Save(ctx context.Context, a *model.InsurancePolicyAggregate) error {
	payer, err := encryptField(ctx, r.cipher, a.PayerIdentifier)
	if err != nil {
		return err
	}
	doc := &insurancePolicyDoc{
		DocID:                     a.ID,
		Ver:                       a.GetVersion(),
		StatusV:                   string(a.Status),
		PatientID:                 a.PatientID,
		PayerIdentifierCipher:     payer,
		EffectiveStart:            a.EffectiveDates.Start,
		EffectiveEnd:              a.EffectiveDates.End,
		VerifiedServiceDate:       a.VerifiedServiceDate,
		EligibilityNotVerified:    a.EligibilityNotVerified,
		ActivePrimaryPolicyExists: a.ActivePrimaryPolicyExists,
		PolicyExpired:             a.PolicyExpired,
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads and decrypts an insurance-policy aggregate by identity.
func (r *InsurancePolicyRepository) FindByID(ctx context.Context, id string) (*model.InsurancePolicyAggregate, error) {
	var doc insurancePolicyDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	payer, err := decryptField(ctx, r.cipher, doc.PayerIdentifierCipher)
	if err != nil {
		return nil, err
	}
	a := &model.InsurancePolicyAggregate{
		ID:                        doc.DocID,
		Status:                    model.PolicyStatus(doc.StatusV),
		PatientID:                 doc.PatientID,
		PayerIdentifier:           payer,
		EffectiveDates:            model.EffectiveDates{Start: doc.EffectiveStart, End: doc.EffectiveEnd},
		VerifiedServiceDate:       doc.VerifiedServiceDate,
		EligibilityNotVerified:    doc.EligibilityNotVerified,
		ActivePrimaryPolicyExists: doc.ActivePrimaryPolicyExists,
		PolicyExpired:             doc.PolicyExpired,
	}
	a.Version = doc.Ver
	return a, nil
}
