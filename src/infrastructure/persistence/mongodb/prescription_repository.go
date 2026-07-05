package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	engagerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// prescriptionsCollection is the collection prescription documents live in.
const prescriptionsCollection = "prescriptions"

// prescriptionDoc is the at-rest projection of a PrescriptionAggregate. The
// medication and dosage are PHI and stored encrypted.
type prescriptionDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	ScopedPatientID  string `bson:"scoped_patient_id"`
	ScopedProviderID string `bson:"scoped_provider_id"`

	MedicationCipher crypto.CipherText `bson:"medication_cipher"`
	DosageCipher     crypto.CipherText `bson:"dosage_cipher"`

	ProviderUnauthorized       bool `bson:"provider_unauthorized"`
	SafetyCheckFailed          bool `bson:"safety_check_failed"`
	SafetyOverrideAcknowledged bool `bson:"safety_override_acknowledged"`
	SafetyChecked              bool `bson:"safety_checked"`
}

func (d *prescriptionDoc) ID() string       { return d.DocID }
func (d *prescriptionDoc) Version() int     { return d.Ver }
func (d *prescriptionDoc) SetVersion(v int) { d.Ver = v }

// PrescriptionRepository is the MongoDB-backed PrescriptionRepository, encrypting
// the prescribed medication and dosage with the S-68 envelope cipher.
type PrescriptionRepository struct {
	base   *BaseRepository
	cipher *crypto.FieldCipher
}

var _ engagerepo.PrescriptionRepository = (*PrescriptionRepository)(nil)

// NewPrescriptionRepository builds a prescription repository over a store and cipher.
func NewPrescriptionRepository(store DocumentStore, cipher *crypto.FieldCipher) *PrescriptionRepository {
	return &PrescriptionRepository{base: NewBaseRepository(store, prescriptionsCollection), cipher: cipher}
}

// Save encrypts the prescription's PHI and persists it with optimistic concurrency.
func (r *PrescriptionRepository) Save(ctx context.Context, a *model.PrescriptionAggregate) error {
	med, err := encryptField(ctx, r.cipher, a.Medication)
	if err != nil {
		return err
	}
	dose, err := encryptField(ctx, r.cipher, a.Dosage)
	if err != nil {
		return err
	}
	doc := &prescriptionDoc{
		DocID:                      a.ID,
		Ver:                        a.GetVersion(),
		StatusV:                    string(a.Status),
		ScopedPatientID:            a.ScopedPatientID,
		ScopedProviderID:           a.ScopedProviderID,
		MedicationCipher:           med,
		DosageCipher:               dose,
		ProviderUnauthorized:       a.ProviderUnauthorized,
		SafetyCheckFailed:          a.SafetyCheckFailed,
		SafetyOverrideAcknowledged: a.SafetyOverrideAcknowledged,
		SafetyChecked:              a.SafetyChecked,
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads and decrypts a prescription aggregate by identity.
func (r *PrescriptionRepository) FindByID(ctx context.Context, id string) (*model.PrescriptionAggregate, error) {
	var doc prescriptionDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	med, err := decryptField(ctx, r.cipher, doc.MedicationCipher)
	if err != nil {
		return nil, err
	}
	dose, err := decryptField(ctx, r.cipher, doc.DosageCipher)
	if err != nil {
		return nil, err
	}
	a := &model.PrescriptionAggregate{
		ID:                         doc.DocID,
		Status:                     model.PrescriptionStatus(doc.StatusV),
		ScopedPatientID:            doc.ScopedPatientID,
		ScopedProviderID:           doc.ScopedProviderID,
		Medication:                 med,
		Dosage:                     dose,
		ProviderUnauthorized:       doc.ProviderUnauthorized,
		SafetyCheckFailed:          doc.SafetyCheckFailed,
		SafetyOverrideAcknowledged: doc.SafetyOverrideAcknowledged,
		SafetyChecked:              doc.SafetyChecked,
	}
	a.Version = doc.Ver
	return a, nil
}
