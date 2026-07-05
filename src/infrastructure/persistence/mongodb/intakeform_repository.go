package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	engagerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// intakeFormsCollection is the collection intake-form documents live in.
const intakeFormsCollection = "intake_forms"

// intakeFormDoc is the at-rest projection of an IntakeFormAggregate. The
// submitted history and demographics are patient PHI and stored encrypted.
type intakeFormDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	ScopedPatientID    string            `bson:"scoped_patient_id"`
	HistoryCipher      crypto.CipherText `bson:"history_cipher"`
	DemographicsCipher crypto.CipherText `bson:"demographics_cipher"`

	Incomplete bool `bson:"incomplete"`
}

func (d *intakeFormDoc) ID() string       { return d.DocID }
func (d *intakeFormDoc) Version() int     { return d.Ver }
func (d *intakeFormDoc) SetVersion(v int) { d.Ver = v }

// IntakeFormRepository is the MongoDB-backed IntakeFormRepository, encrypting the
// submitted history and demographics with the S-68 envelope cipher.
type IntakeFormRepository struct {
	base   *BaseRepository
	cipher *crypto.FieldCipher
}

var _ engagerepo.IntakeFormRepository = (*IntakeFormRepository)(nil)

// NewIntakeFormRepository builds an intake-form repository over a store and cipher.
func NewIntakeFormRepository(store DocumentStore, cipher *crypto.FieldCipher) *IntakeFormRepository {
	return &IntakeFormRepository{base: NewBaseRepository(store, intakeFormsCollection), cipher: cipher}
}

// Save encrypts the intake PHI and persists it with optimistic concurrency.
func (r *IntakeFormRepository) Save(ctx context.Context, a *model.IntakeFormAggregate) error {
	history, err := encryptField(ctx, r.cipher, a.History)
	if err != nil {
		return err
	}
	demographics, err := encryptField(ctx, r.cipher, a.Demographics)
	if err != nil {
		return err
	}
	doc := &intakeFormDoc{
		DocID:              a.ID,
		Ver:                a.GetVersion(),
		StatusV:            string(a.Status),
		ScopedPatientID:    a.ScopedPatientID,
		HistoryCipher:      history,
		DemographicsCipher: demographics,
		Incomplete:         a.Incomplete,
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads and decrypts an intake-form aggregate by identity.
func (r *IntakeFormRepository) FindByID(ctx context.Context, id string) (*model.IntakeFormAggregate, error) {
	var doc intakeFormDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	history, err := decryptField(ctx, r.cipher, doc.HistoryCipher)
	if err != nil {
		return nil, err
	}
	demographics, err := decryptField(ctx, r.cipher, doc.DemographicsCipher)
	if err != nil {
		return nil, err
	}
	a := &model.IntakeFormAggregate{
		ID:              doc.DocID,
		Status:          model.IntakeFormStatus(doc.StatusV),
		ScopedPatientID: doc.ScopedPatientID,
		History:         history,
		Demographics:    demographics,
		Incomplete:      doc.Incomplete,
	}
	a.Version = doc.Ver
	return a, nil
}
