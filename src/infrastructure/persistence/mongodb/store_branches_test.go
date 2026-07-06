package mongodb

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	authzmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	bill "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	clin "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	identitymodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/model"
	engage "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	sched "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
)

// TestMemStore_Branches covers the MemStore branches the higher-level repository
// tests do not exercise directly: a rejected duplicate insert and a
// version-guarded replace against a missing document.
func TestMemStore_Branches(t *testing.T) {
	ctx := context.Background()
	m := NewMemStore()

	if err := m.InsertOne(ctx, "w-1", &widget{WID: "w-1", Ver: 1}); err != nil {
		t.Fatalf("first InsertOne: %v", err)
	}
	if err := m.InsertOne(ctx, "w-1", &widget{WID: "w-1", Ver: 1}); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("duplicate InsertOne = %v, want ErrDuplicateKey", err)
	}

	// Replacing a document that was never stored is a guard miss, not an error.
	matched, err := m.ReplaceVersioned(ctx, "missing", 1, &widget{WID: "missing", Ver: 1})
	if err != nil || matched {
		t.Fatalf("ReplaceVersioned(missing) = (%v, %v), want (false, nil)", matched, err)
	}
}

// foundStore reports every id as an already-stored document at a fixed version,
// letting the encrypted store's update path (and its version-guard-miss branch)
// be driven deterministically without a live database.
type foundStore struct {
	version    int
	matched    bool
	replaceErr error
}

func (s foundStore) InsertOne(context.Context, string, any) error { return nil }
func (s foundStore) FindOne(_ context.Context, _ string, dest any) error {
	if m, ok := dest.(*bson.M); ok {
		*m = bson.M{versionField: s.version}
	}
	return nil
}
func (s foundStore) ReplaceVersioned(context.Context, string, int, any) (bool, error) {
	return s.matched, s.replaceErr
}
func (s foundStore) DeleteOne(context.Context, string) (bool, error) { return true, nil }

// TestEncryptedStore_UpdateConflict covers encryptedStore.save's update path:
// the stored version is read, the replace fails the guard, and a typed
// OptimisticConcurrencyError (wrapping the shared sentinel) is returned.
func TestEncryptedStore_UpdateConflict(t *testing.T) {
	repo := NewUserAccountRepository(foundStore{version: 5, matched: false}, newTestCodec(t))
	err := repo.Save(context.Background(), registeredAccount(t, "user-c", "c@ancora.health"))

	var occ *OptimisticConcurrencyError
	if !errors.As(err, &occ) {
		t.Fatalf("expected *OptimisticConcurrencyError, got %T: %v", err, err)
	}
	if occ.ExpectedVersion != 5 {
		t.Fatalf("ExpectedVersion = %d, want 5", occ.ExpectedVersion)
	}
}

// TestEncryptedStore_UpdateReplaceError covers the replace-error branch of the
// encrypted store's update path.
func TestEncryptedStore_UpdateReplaceError(t *testing.T) {
	repo := NewUserAccountRepository(foundStore{version: 5, matched: false, replaceErr: errBoom}, newTestCodec(t))
	if err := repo.Save(context.Background(), registeredAccount(t, "user-e", "e@ancora.health")); !errors.Is(err, errBoom) {
		t.Fatalf("Save error = %v, want errBoom", err)
	}
}

// TestUpsert_UpdatePath covers the plain (non-encrypted) upsert update branch: a
// second Save of an already-stored session finds the current version and issues
// a version-guarded replace.
func TestUpsert_UpdatePath(t *testing.T) {
	ctx := context.Background()
	repo := NewSessionRepository(NewMemStore())

	sess := &identitymodel.SessionAggregate{ID: "sess-u", Issued: true, AccountID: "acct-1"}
	if err := repo.Save(ctx, sess); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	// Second save takes the upsert update path (found → version-guarded replace).
	if err := repo.Save(ctx, sess); err != nil {
		t.Fatalf("second Save (update): %v", err)
	}
	got, err := repo.FindByID(ctx, "sess-u")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.AccountID != "acct-1" {
		t.Fatalf("account mismatch after update: %+v", got)
	}
}

// TestSaveAggregate_UpdatePath covers saveAggregate's update branch. After a
// command bumps the aggregate to version 1 (with one buffered event), the first
// Save inserts and clears the event; the aggregate then reports a load-time
// version of 1, so the second Save goes through the version-guarded replace
// rather than an insert.
func TestSaveAggregate_UpdatePath(t *testing.T) {
	ctx := context.Background()
	repo := NewAppointmentRepository(NewMemStore(), NewMemStore(), locking.NewMemorySlotLocker(), "")

	a := &sched.AppointmentAggregate{ID: "appt-u"}
	if _, err := a.Execute(sched.HoldSlotCmd{ProviderId: "prov-1", TimeSlot: "2026-07-10T09:00", PatientId: "pat-1"}); err != nil {
		t.Fatalf("HoldSlot: %v", err)
	}
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("first Save (insert): %v", err)
	}
	// The aggregate is now at version 1 with no buffered events, so this Save takes
	// the update branch.
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("second Save (update): %v", err)
	}
	got, err := repo.FindByID(ctx, "appt-u")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.GetVersion() < 2 {
		t.Fatalf("expected a persisted version >= 2 after update, got %d", got.GetVersion())
	}
}

// failAuditColl fails both operations, driving the audit repository's Load-error
// branches on Save and FindByID.
type failAuditColl struct{ err error }

func (c failAuditColl) Append(context.Context, auditEntryDoc) error { return c.err }
func (c failAuditColl) Load(context.Context, string) ([]auditEntryDoc, error) {
	return nil, c.err
}

// appendFailColl loads cleanly (nothing stored) but fails on append, driving the
// non-duplicate append-error branch of Save.
type appendFailColl struct{ err error }

func (c appendFailColl) Append(context.Context, auditEntryDoc) error { return c.err }
func (c appendFailColl) Load(context.Context, string) ([]auditEntryDoc, error) {
	return nil, nil
}

// TestAuditTrail_ErrorPaths covers the audit repository and in-memory collection
// branches the round-trip tests miss: a not-found rehydrate, a rejected duplicate
// append, and both collection-failure branches of Save.
func TestAuditTrail_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("find not found", func(t *testing.T) {
		repo := NewAuditTrailRepository(NewMemAuditEntryCollection())
		if _, err := repo.FindByID(ctx, "absent"); !errors.Is(err, ErrDocumentNotFound) {
			t.Fatalf("FindByID = %v, want ErrDocumentNotFound", err)
		}
	})

	t.Run("mem duplicate append", func(t *testing.T) {
		c := NewMemAuditEntryCollection()
		doc := auditEntryDoc{DocID: "trail#1", TrailID: "trail", Sequence: 1}
		if err := c.Append(ctx, doc); err != nil {
			t.Fatalf("first Append: %v", err)
		}
		if err := c.Append(ctx, doc); !errors.Is(err, ErrDuplicateKey) {
			t.Fatalf("duplicate Append = %v, want ErrDuplicateKey", err)
		}
	})

	t.Run("save load error", func(t *testing.T) {
		repo := NewAuditTrailRepository(failAuditColl{err: errBoom})
		if err := repo.Save(ctx, appendableTrail(t, "trail-le", 1)); !errors.Is(err, errBoom) {
			t.Fatalf("Save error = %v, want errBoom", err)
		}
	})

	t.Run("find load error", func(t *testing.T) {
		repo := NewAuditTrailRepository(failAuditColl{err: errBoom})
		if _, err := repo.FindByID(ctx, "trail-le"); !errors.Is(err, errBoom) {
			t.Fatalf("FindByID error = %v, want errBoom", err)
		}
	})

	t.Run("save append error", func(t *testing.T) {
		repo := NewAuditTrailRepository(appendFailColl{err: errBoom})
		if err := repo.Save(ctx, appendableTrail(t, "trail-ae", 1)); !errors.Is(err, errBoom) {
			t.Fatalf("Save error = %v, want errBoom", err)
		}
	})
}

// TestEncryptedRepositories_WrongKeyDecryptFails covers the decryption-error
// branch on the FindByID path of the field-encrypting repositories: a document
// sealed under one key cannot be read back with a different key.
func TestEncryptedRepositories_WrongKeyDecryptFails(t *testing.T) {
	ctx := context.Background()

	// A second cipher whose key genuinely differs from the all-zero test key, so
	// reads with it fail to decrypt.
	altKey := make([]byte, crypto.KeySize)
	altKey[0] = 0x5c
	altEnv, err := crypto.NewAESKeyEnvelope("mismatch", altKey)
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope alt: %v", err)
	}
	altCipher := crypto.NewFieldCipher(altEnv)

	cases := []struct {
		name  string
		write func(store DocumentStore)
		read  func(store DocumentStore) error
	}{
		{
			"intakeform",
			func(store DocumentStore) {
				repo := NewIntakeFormRepository(store, testCipher(t))
				_ = repo.Save(ctx, &engage.IntakeFormAggregate{ID: "x", ScopedPatientID: "p", History: "hx", Demographics: "dg"})
			},
			func(store DocumentStore) error {
				_, err := NewIntakeFormRepository(store, altCipher).FindByID(ctx, "x")
				return err
			},
		},
		{
			"laborder",
			func(store DocumentStore) {
				repo := NewLabOrderRepository(store, testCipher(t))
				_ = repo.Save(ctx, &clin.LabOrderAggregate{ID: "x", ScopedPatientID: "p", TestCode: "CBC"})
			},
			func(store DocumentStore) error {
				_, err := NewLabOrderRepository(store, altCipher).FindByID(ctx, "x")
				return err
			},
		},
		{
			"insurancepolicy",
			func(store DocumentStore) {
				repo := NewInsurancePolicyRepository(store, testCipher(t))
				_ = repo.Save(ctx, &bill.InsurancePolicyAggregate{ID: "x", PatientID: "p", PayerIdentifier: "AETNA"})
			},
			func(store DocumentStore) error {
				_, err := NewInsurancePolicyRepository(store, altCipher).FindByID(ctx, "x")
				return err
			},
		},
		{
			"payment",
			func(store DocumentStore) {
				repo := NewPaymentRepository(store, testCipher(t))
				_ = repo.Save(ctx, &bill.PaymentAggregate{ID: "x", InvoiceID: "inv", PaymentToken: "tok"})
			},
			func(store DocumentStore) error {
				_, err := NewPaymentRepository(store, altCipher).FindByID(ctx, "x")
				return err
			},
		},
		{
			"prescription",
			func(store DocumentStore) {
				repo := NewPrescriptionRepository(store, testCipher(t))
				_ = repo.Save(ctx, &engage.PrescriptionAggregate{ID: "x", Medication: "amox", Dosage: "500mg"})
			},
			func(store DocumentStore) error {
				_, err := NewPrescriptionRepository(store, altCipher).FindByID(ctx, "x")
				return err
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := NewMemStore()
			c.write(store)
			if err := c.read(store); err == nil {
				t.Fatal("expected a decryption failure reading under a mismatched key")
			}
		})
	}
}

// brokenEnvelope is a KeyEnvelope whose wrap/unwrap always fail, so a
// FieldCipher (or Codec) built over it makes every encrypt attempt error. It
// drives the encrypt-error branches the repositories guard but a healthy cipher
// never triggers.
type brokenEnvelope struct{}

func (brokenEnvelope) WrapDataKey(context.Context, []byte) ([]byte, string, error) {
	return nil, "", errBoom
}
func (brokenEnvelope) UnwrapDataKey(context.Context, []byte, string) ([]byte, error) {
	return nil, errBoom
}

// TestSave_EncryptFailurePropagates covers the encrypt-error guard on every
// field-encrypting and codec-encrypting repository: when sealing PHI fails, Save
// surfaces the error rather than persisting plaintext.
func TestSave_EncryptFailurePropagates(t *testing.T) {
	ctx := context.Background()
	broken := crypto.NewFieldCipher(brokenEnvelope{})
	brokenCodec := crypto.NewCodec(broken)

	cases := []struct {
		name string
		save func() error
	}{
		{"encounter", func() error {
			return NewEncounterRepository(NewMemStore(), broken).Save(ctx, &clin.EncounterAggregate{
				ID:        "e",
				Note:      &clin.ClinicalNote{Content: "phi"},
				Diagnoses: []clin.Diagnosis{{Code: "c", Description: "d"}},
				Addenda:   []clin.Addendum{{Text: "t", AuthorID: "a"}},
			})
		}},
		{"encounter diagnosis only", func() error {
			return NewEncounterRepository(NewMemStore(), broken).Save(ctx, &clin.EncounterAggregate{
				ID:        "e-d",
				Diagnoses: []clin.Diagnosis{{Code: "c", Description: "d"}},
			})
		}},
		{"encounter addendum only", func() error {
			return NewEncounterRepository(NewMemStore(), broken).Save(ctx, &clin.EncounterAggregate{
				ID:      "e-a",
				Addenda: []clin.Addendum{{Text: "t", AuthorID: "a"}},
			})
		}},
		{"intakeform", func() error {
			return NewIntakeFormRepository(NewMemStore(), broken).Save(ctx, &engage.IntakeFormAggregate{ID: "i", History: "h", Demographics: "d"})
		}},
		{"insurancepolicy", func() error {
			return NewInsurancePolicyRepository(NewMemStore(), broken).Save(ctx, &bill.InsurancePolicyAggregate{ID: "p", PayerIdentifier: "X"})
		}},
		{"laborder", func() error {
			return NewLabOrderRepository(NewMemStore(), broken).Save(ctx, &clin.LabOrderAggregate{ID: "l", TestCode: "CBC"})
		}},
		{"payment", func() error {
			return NewPaymentRepository(NewMemStore(), broken).Save(ctx, &bill.PaymentAggregate{ID: "p", PaymentToken: "tok"})
		}},
		{"prescription", func() error {
			return NewPrescriptionRepository(NewMemStore(), broken).Save(ctx, &engage.PrescriptionAggregate{ID: "rx", Medication: "m", Dosage: "d"})
		}},
		{"useraccount", func() error {
			return NewUserAccountRepository(NewMemStore(), brokenCodec).Save(ctx, registeredAccount(t, "u", "u@ancora.health"))
		}},
		{"authorizationpolicy", func() error {
			p := &authzmodel.AuthorizationPolicyAggregate{ID: "pol"}
			_, _ = p.Execute(authzmodel.PublishPolicyVersionCmd{RegoBundle: "package a", Author: "dr@x"})
			return NewAuthorizationPolicyRepository(NewMemStore(), brokenCodec).Save(ctx, p)
		}},
		{"carerelationship", func() error {
			r := &authzmodel.CareRelationshipAggregate{ID: "rel"}
			_, _ = r.Execute(authzmodel.EstablishCareRelationshipCmd{ProviderID: "pr", PatientID: "pt", ClinicID: "cl"})
			return NewCareRelationshipRepository(NewMemStore(), brokenCodec).Save(ctx, r)
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.save(); err == nil {
				t.Fatal("expected an encryption failure to propagate from Save")
			}
		})
	}
}

// TestEncryptedStore_UpdateEncryptError covers the encrypt-error branch on the
// encrypted store's update path: the document is found, but re-sealing it fails.
func TestEncryptedStore_UpdateEncryptError(t *testing.T) {
	brokenCodec := crypto.NewCodec(crypto.NewFieldCipher(brokenEnvelope{}))
	repo := NewUserAccountRepository(foundStore{version: 2, matched: true}, brokenCodec)
	if err := repo.Save(context.Background(), registeredAccount(t, "u2", "u2@ancora.health")); !errors.Is(err, errBoom) {
		t.Fatalf("Save error = %v, want errBoom", err)
	}
}

// dupAppendColl loads cleanly but reports a duplicate on append, driving the
// append-only immutability branch of the audit repository's Save.
type dupAppendColl struct{}

func (dupAppendColl) Append(context.Context, auditEntryDoc) error { return ErrDuplicateKey }
func (dupAppendColl) Load(context.Context, string) ([]auditEntryDoc, error) {
	return nil, nil
}

// TestAuditTrail_Save_DuplicateIsImmutable covers the branch that maps a
// storage-layer duplicate-key collision to the domain's append-only violation.
func TestAuditTrail_Save_DuplicateIsImmutable(t *testing.T) {
	repo := NewAuditTrailRepository(dupAppendColl{})
	if err := repo.Save(context.Background(), appendableTrail(t, "trail-d", 1)); !errors.Is(err, auditmodel.ErrAuditEntryImmutable) {
		t.Fatalf("Save error = %v, want ErrAuditEntryImmutable", err)
	}
}

// TestEncounterRepository_RoundTrip_WithAddenda covers the addenda mapping in
// both toDoc and fromDoc, which the existing encounter test (note + diagnosis
// only) does not reach.
func TestEncounterRepository_RoundTrip_WithAddenda(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewEncounterRepository(store, testCipher(t))

	enc := &clin.EncounterAggregate{
		ID:      "enc-add",
		Status:  clin.EncounterStatus("open"),
		Addenda: []clin.Addendum{{Text: "late correction", AuthorID: "dr-1"}},
	}
	if err := repo.Save(ctx, enc); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.FindByID(ctx, "enc-add")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if len(got.Addenda) != 1 || got.Addenda[0].Text != "late correction" || got.Addenda[0].AuthorID != "dr-1" {
		t.Fatalf("addenda did not round-trip: %+v", got.Addenda)
	}
}

// TestEncounterRepository_FindByID_DiagnosisAndAddendumDecryptErrors covers the
// fromDoc decryption-error branches for a diagnosis description and an addendum
// body (each without a Note, so decryption reaches those loops before failing).
func TestEncounterRepository_FindByID_DiagnosisAndAddendumDecryptErrors(t *testing.T) {
	ctx := context.Background()
	altKey := make([]byte, crypto.KeySize)
	altKey[0] = 0x33
	altEnv, err := crypto.NewAESKeyEnvelope("alt", altKey)
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	altCipher := crypto.NewFieldCipher(altEnv)

	t.Run("diagnosis", func(t *testing.T) {
		store := NewMemStore()
		_ = NewEncounterRepository(store, testCipher(t)).Save(ctx, &clin.EncounterAggregate{
			ID:        "enc-d",
			Diagnoses: []clin.Diagnosis{{Code: "I20.9", Description: "angina"}},
		})
		if _, err := NewEncounterRepository(store, altCipher).FindByID(ctx, "enc-d"); err == nil {
			t.Fatal("expected a diagnosis decryption failure")
		}
	})

	t.Run("addendum", func(t *testing.T) {
		store := NewMemStore()
		_ = NewEncounterRepository(store, testCipher(t)).Save(ctx, &clin.EncounterAggregate{
			ID:      "enc-a",
			Addenda: []clin.Addendum{{Text: "note", AuthorID: "dr-1"}},
		})
		if _, err := NewEncounterRepository(store, altCipher).FindByID(ctx, "enc-a"); err == nil {
			t.Fatal("expected an addendum decryption failure")
		}
	})
}
