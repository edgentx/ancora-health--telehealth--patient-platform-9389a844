package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	clinrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// encountersCollection is the collection encounter documents live in.
const encountersCollection = "encounters"

// encounterDoc is the at-rest projection of an EncounterAggregate. The clinical
// narrative — the SOAP note body, each diagnosis description and each addendum
// text — is PHI and is stored as CipherText, never as plaintext.
type encounterDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	ScopedProviderID string `bson:"scoped_provider_id"`
	ScopedPatientID  string `bson:"scoped_patient_id"`
	VideoRoomID      string `bson:"video_room_id"`

	Note      *clinicalNoteDoc `bson:"note,omitempty"`
	Diagnoses []diagnosisDoc   `bson:"diagnoses"`
	Addenda   []addendumDoc    `bson:"addenda"`
}

// clinicalNoteDoc stores a SOAP note with its PHI body encrypted.
type clinicalNoteDoc struct {
	ContentCipher crypto.CipherText `bson:"content_cipher"`
	Signed        bool              `bson:"signed"`
}

// diagnosisDoc stores a coded diagnosis; the human-readable description is PHI
// and encrypted, while the terminology code is retained in the clear for
// indexing and rollups.
type diagnosisDoc struct {
	Code              string            `bson:"code"`
	DescriptionCipher crypto.CipherText `bson:"description_cipher"`
}

// addendumDoc stores a note correction with its PHI body encrypted.
type addendumDoc struct {
	TextCipher crypto.CipherText `bson:"text_cipher"`
	AuthorID   string            `bson:"author_id"`
}

func (d *encounterDoc) ID() string       { return d.DocID }
func (d *encounterDoc) Version() int     { return d.Ver }
func (d *encounterDoc) SetVersion(v int) { d.Ver = v }

// EncounterRepository is the MongoDB-backed EncounterRepository. It encrypts the
// encounter's PHI fields with the S-68 envelope cipher on the way to storage and
// decrypts them on the way back.
type EncounterRepository struct {
	base   *BaseRepository
	cipher *crypto.FieldCipher
}

var _ clinrepo.EncounterRepository = (*EncounterRepository)(nil)

// NewEncounterRepository builds an encounter repository over a store and cipher.
func NewEncounterRepository(store DocumentStore, cipher *crypto.FieldCipher) *EncounterRepository {
	return &EncounterRepository{base: NewBaseRepository(store, encountersCollection), cipher: cipher}
}

// Save encrypts the encounter's PHI and persists it with optimistic concurrency.
func (r *EncounterRepository) Save(ctx context.Context, a *model.EncounterAggregate) error {
	doc, err := r.toDoc(ctx, a)
	if err != nil {
		return err
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads and decrypts an encounter aggregate by identity.
func (r *EncounterRepository) FindByID(ctx context.Context, id string) (*model.EncounterAggregate, error) {
	var doc encounterDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	return r.fromDoc(ctx, &doc)
}

func (r *EncounterRepository) toDoc(ctx context.Context, a *model.EncounterAggregate) (*encounterDoc, error) {
	doc := &encounterDoc{
		DocID:            a.ID,
		Ver:              a.GetVersion(),
		StatusV:          string(a.Status),
		ScopedProviderID: a.ScopedProviderID,
		ScopedPatientID:  a.ScopedPatientID,
		VideoRoomID:      a.VideoRoomID,
	}

	if a.Note != nil {
		ct, err := encryptField(ctx, r.cipher, a.Note.Content)
		if err != nil {
			return nil, err
		}
		doc.Note = &clinicalNoteDoc{ContentCipher: ct, Signed: a.Note.Signed}
	}

	for _, d := range a.Diagnoses {
		ct, err := encryptField(ctx, r.cipher, d.Description)
		if err != nil {
			return nil, err
		}
		doc.Diagnoses = append(doc.Diagnoses, diagnosisDoc{Code: d.Code, DescriptionCipher: ct})
	}

	for _, ad := range a.Addenda {
		ct, err := encryptField(ctx, r.cipher, ad.Text)
		if err != nil {
			return nil, err
		}
		doc.Addenda = append(doc.Addenda, addendumDoc{TextCipher: ct, AuthorID: ad.AuthorID})
	}

	return doc, nil
}

func (r *EncounterRepository) fromDoc(ctx context.Context, d *encounterDoc) (*model.EncounterAggregate, error) {
	a := &model.EncounterAggregate{
		ID:               d.DocID,
		Status:           model.EncounterStatus(d.StatusV),
		ScopedProviderID: d.ScopedProviderID,
		ScopedPatientID:  d.ScopedPatientID,
		VideoRoomID:      d.VideoRoomID,
	}
	a.Version = d.Ver

	if d.Note != nil {
		content, err := decryptField(ctx, r.cipher, d.Note.ContentCipher)
		if err != nil {
			return nil, err
		}
		a.Note = &model.ClinicalNote{Content: content, Signed: d.Note.Signed}
	}

	for _, dd := range d.Diagnoses {
		desc, err := decryptField(ctx, r.cipher, dd.DescriptionCipher)
		if err != nil {
			return nil, err
		}
		a.Diagnoses = append(a.Diagnoses, model.Diagnosis{Code: dd.Code, Description: desc})
	}

	for _, ad := range d.Addenda {
		text, err := decryptField(ctx, r.cipher, ad.TextCipher)
		if err != nil {
			return nil, err
		}
		a.Addenda = append(a.Addenda, model.Addendum{Text: text, AuthorID: ad.AuthorID})
	}

	return a, nil
}
