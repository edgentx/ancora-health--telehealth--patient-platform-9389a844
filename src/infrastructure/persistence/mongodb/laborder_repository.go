package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	clinrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// labOrdersCollection is the collection lab-order documents live in.
const labOrdersCollection = "lab_orders"

// labOrderDoc is the at-rest projection of a LabOrderAggregate. The ordered test
// code, which reveals the clinical purpose of the order for an identified
// patient, is PHI and stored encrypted.
type labOrderDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	ScopedPatientID  string `bson:"scoped_patient_id"`
	ScopedProviderID string `bson:"scoped_provider_id"`

	CareRelationshipActive bool              `bson:"care_relationship_active"`
	TestCodeCipher         crypto.CipherText `bson:"test_code_cipher"`
}

func (d *labOrderDoc) ID() string       { return d.DocID }
func (d *labOrderDoc) Version() int     { return d.Ver }
func (d *labOrderDoc) SetVersion(v int) { d.Ver = v }

// LabOrderRepository is the MongoDB-backed LabOrderRepository, encrypting the
// order's PHI field with the S-68 envelope cipher.
type LabOrderRepository struct {
	base   *BaseRepository
	cipher *crypto.FieldCipher
}

var _ clinrepo.LabOrderRepository = (*LabOrderRepository)(nil)

// NewLabOrderRepository builds a lab-order repository over a store and cipher.
func NewLabOrderRepository(store DocumentStore, cipher *crypto.FieldCipher) *LabOrderRepository {
	return &LabOrderRepository{base: NewBaseRepository(store, labOrdersCollection), cipher: cipher}
}

// Save encrypts the order's PHI and persists it with optimistic concurrency.
func (r *LabOrderRepository) Save(ctx context.Context, a *model.LabOrderAggregate) error {
	ct, err := encryptField(ctx, r.cipher, a.TestCode)
	if err != nil {
		return err
	}
	doc := &labOrderDoc{
		DocID:                  a.ID,
		Ver:                    a.GetVersion(),
		StatusV:                string(a.Status),
		ScopedPatientID:        a.ScopedPatientID,
		ScopedProviderID:       a.ScopedProviderID,
		CareRelationshipActive: a.CareRelationshipActive,
		TestCodeCipher:         ct,
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads and decrypts a lab-order aggregate by identity.
func (r *LabOrderRepository) FindByID(ctx context.Context, id string) (*model.LabOrderAggregate, error) {
	var doc labOrderDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	testCode, err := decryptField(ctx, r.cipher, doc.TestCodeCipher)
	if err != nil {
		return nil, err
	}
	a := &model.LabOrderAggregate{
		ID:                     doc.DocID,
		Status:                 model.LabOrderStatus(doc.StatusV),
		ScopedPatientID:        doc.ScopedPatientID,
		ScopedProviderID:       doc.ScopedProviderID,
		CareRelationshipActive: doc.CareRelationshipActive,
		TestCode:               testCode,
	}
	a.Version = doc.Ver
	return a, nil
}
