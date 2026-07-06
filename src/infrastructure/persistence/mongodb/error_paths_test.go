package mongodb

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	admin "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
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

// errBoom is the generic infrastructure failure the fakes surface.
var errBoom = errors.New("mongodb-test: store boom")

// errStore is a DocumentStore whose every operation fails with a fixed error. It
// drives the store-error propagation branches the in-memory MemStore never hits.
type errStore struct{ err error }

func (s errStore) InsertOne(context.Context, string, any) error { return s.err }
func (s errStore) FindOne(context.Context, string, any) error   { return s.err }
func (s errStore) ReplaceVersioned(context.Context, string, int, any) (bool, error) {
	return false, s.err
}
func (s errStore) DeleteOne(context.Context, string) (bool, error) { return false, s.err }

// fakeLocker is a configurable SlotLocker for exercising Book's non-happy paths.
type fakeLocker struct {
	acquired bool
	acqErr   error
}

func (l fakeLocker) Acquire(context.Context, string, string, time.Duration) (bool, error) {
	return l.acquired, l.acqErr
}
func (l fakeLocker) Release(context.Context, string, string) error { return nil }

// errFactSource is a FactSource whose queries fail, driving the rollup error
// branches.
type errFactSource struct{ err error }

func (f errFactSource) AppointmentFacts(context.Context, string, time.Time, time.Time) ([]AppointmentFact, error) {
	return nil, f.err
}
func (f errFactSource) RevenueFacts(context.Context, string, time.Time, time.Time) ([]RevenueFact, error) {
	return nil, f.err
}

// TestRepositories_StoreErrorPropagation asserts that every repository surfaces a
// store failure from both its Save and its FindByID entrypoints. Each repository
// routes writes through insert / version-guarded-replace and reads through
// FindOne, so a failing store must propagate the error unchanged.
func TestRepositories_StoreErrorPropagation(t *testing.T) {
	ctx := context.Background()
	codec := newTestCodec(t)
	cipher := testCipher(t)
	locker := locking.NewMemorySlotLocker()

	type repoCase struct {
		name string
		save func(DocumentStore) error
		find func(DocumentStore) error
	}
	cases := []repoCase{
		{
			"appointment",
			func(s DocumentStore) error {
				return NewAppointmentRepository(s, NewMemStore(), locker, "").Save(ctx, &sched.AppointmentAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewAppointmentRepository(s, NewMemStore(), locker, "").FindByID(ctx, "a")
				return err
			},
		},
		{
			"encounter",
			func(s DocumentStore) error {
				return NewEncounterRepository(s, cipher).Save(ctx, &clin.EncounterAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewEncounterRepository(s, cipher).FindByID(ctx, "a")
				return err
			},
		},
		{
			"useraccount",
			func(s DocumentStore) error {
				return NewUserAccountRepository(s, codec).Save(ctx, &identitymodel.UserAccountAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewUserAccountRepository(s, codec).FindByID(ctx, "a")
				return err
			},
		},
		{
			"session",
			func(s DocumentStore) error {
				return NewSessionRepository(s).Save(ctx, &identitymodel.SessionAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewSessionRepository(s).FindByID(ctx, "a")
				return err
			},
		},
		{
			"authorizationpolicy",
			func(s DocumentStore) error {
				return NewAuthorizationPolicyRepository(s, codec).Save(ctx, &authzmodel.AuthorizationPolicyAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewAuthorizationPolicyRepository(s, codec).FindByID(ctx, "a")
				return err
			},
		},
		{
			"carerelationship",
			func(s DocumentStore) error {
				return NewCareRelationshipRepository(s, codec).Save(ctx, &authzmodel.CareRelationshipAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewCareRelationshipRepository(s, codec).FindByID(ctx, "a")
				return err
			},
		},
		{
			"cryptokeyenvelope",
			func(s DocumentStore) error {
				return NewCryptoKeyEnvelopeRepository(s).Save(ctx, &auditmodel.CryptoKeyEnvelopeAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewCryptoKeyEnvelopeRepository(s).FindByID(ctx, "a")
				return err
			},
		},
		{
			"clinicdirectory",
			func(s DocumentStore) error {
				return NewClinicDirectoryRepository(s).Save(ctx, &admin.ClinicDirectoryAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewClinicDirectoryRepository(s).FindByID(ctx, "a")
				return err
			},
		},
		{
			"analyticsdashboard",
			func(s DocumentStore) error {
				return NewAnalyticsDashboardRepository(s, nil).Save(ctx, &admin.AnalyticsDashboardAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewAnalyticsDashboardRepository(s, nil).FindByID(ctx, "a")
				return err
			},
		},
		{
			"insurancepolicy",
			func(s DocumentStore) error {
				return NewInsurancePolicyRepository(s, cipher).Save(ctx, &bill.InsurancePolicyAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewInsurancePolicyRepository(s, cipher).FindByID(ctx, "a")
				return err
			},
		},
		{
			"intakeform",
			func(s DocumentStore) error {
				return NewIntakeFormRepository(s, cipher).Save(ctx, &engage.IntakeFormAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewIntakeFormRepository(s, cipher).FindByID(ctx, "a")
				return err
			},
		},
		{
			"invoice",
			func(s DocumentStore) error {
				return NewInvoiceRepository(s).Save(ctx, &bill.InvoiceAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewInvoiceRepository(s).FindByID(ctx, "a")
				return err
			},
		},
		{
			"laborder",
			func(s DocumentStore) error {
				return NewLabOrderRepository(s, cipher).Save(ctx, &clin.LabOrderAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewLabOrderRepository(s, cipher).FindByID(ctx, "a")
				return err
			},
		},
		{
			"messagethread",
			func(s DocumentStore) error {
				return NewMessageThreadRepository(s).Save(ctx, &engage.MessageThreadAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewMessageThreadRepository(s).FindByID(ctx, "a")
				return err
			},
		},
		{
			"payment",
			func(s DocumentStore) error {
				return NewPaymentRepository(s, cipher).Save(ctx, &bill.PaymentAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewPaymentRepository(s, cipher).FindByID(ctx, "a")
				return err
			},
		},
		{
			"prescription",
			func(s DocumentStore) error {
				return NewPrescriptionRepository(s, cipher).Save(ctx, &engage.PrescriptionAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewPrescriptionRepository(s, cipher).FindByID(ctx, "a")
				return err
			},
		},
		{
			"providerschedule",
			func(s DocumentStore) error {
				return NewProviderScheduleRepository(s).Save(ctx, &sched.ProviderScheduleAggregate{ID: "a"})
			},
			func(s DocumentStore) error {
				_, err := NewProviderScheduleRepository(s).FindByID(ctx, "a")
				return err
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name+"/save", func(t *testing.T) {
			if err := c.save(errStore{errBoom}); !errors.Is(err, errBoom) {
				t.Fatalf("Save error = %v, want errBoom", err)
			}
		})
		t.Run(c.name+"/find", func(t *testing.T) {
			if err := c.find(errStore{errBoom}); !errors.Is(err, errBoom) {
				t.Fatalf("FindByID error = %v, want errBoom", err)
			}
		})
	}
}

// TestBaseRepository_Update_StoreError covers the store-error branch of Update
// (distinct from a version-guard miss, which yields the typed conflict).
func TestBaseRepository_Update_StoreError(t *testing.T) {
	repo := NewBaseRepository(errStore{errBoom}, "widgets")
	if err := repo.Update(context.Background(), &widget{WID: "w", Ver: 1}); !errors.Is(err, errBoom) {
		t.Fatalf("Update error = %v, want errBoom", err)
	}
}

// TestBaseRepository_Delete_StoreError covers the store-error branch of Delete.
func TestBaseRepository_Delete_StoreError(t *testing.T) {
	repo := NewBaseRepository(errStore{errBoom}, "widgets")
	if err := repo.Delete(context.Background(), "w"); !errors.Is(err, errBoom) {
		t.Fatalf("Delete error = %v, want errBoom", err)
	}
}

// TestAppointmentRepository_Book_LockerError covers Book's two lock-acquisition
// failure branches: a non-conflict infrastructure error is propagated, while a
// failed (but error-free) acquisition surfaces the typed double-booking conflict.
func TestAppointmentRepository_Book_LockerError(t *testing.T) {
	ctx := context.Background()

	t.Run("infrastructure error", func(t *testing.T) {
		repo := NewAppointmentRepository(NewMemStore(), NewMemStore(), fakeLocker{acquired: false, acqErr: errBoom}, "")
		err := repo.Book(ctx, &sched.AppointmentAggregate{ID: "a"})
		if !errors.Is(err, errBoom) {
			t.Fatalf("Book error = %v, want errBoom", err)
		}
		if errors.Is(err, sched.ErrSlotDoubleBooked) {
			t.Fatal("an infrastructure error must not be reported as a double-booking")
		}
	})

	t.Run("acquisition denied", func(t *testing.T) {
		repo := NewAppointmentRepository(NewMemStore(), NewMemStore(), fakeLocker{acquired: false, acqErr: nil}, "")
		if err := repo.Book(ctx, &sched.AppointmentAggregate{ID: "a"}); !errors.Is(err, sched.ErrSlotDoubleBooked) {
			t.Fatalf("Book error = %v, want ErrSlotDoubleBooked", err)
		}
	})
}

// TestAppointmentRepository_Book_PersistenceError covers Book's transaction
// failure path: the slot lock is acquired, the persistence inside the
// transaction fails, and the error is propagated (after the hold is released).
func TestAppointmentRepository_Book_PersistenceError(t *testing.T) {
	ctx := context.Background()
	repo := NewAppointmentRepository(errStore{errBoom}, NewMemStore(), locking.NewMemorySlotLocker(), "")
	err := repo.Book(ctx, &sched.AppointmentAggregate{ID: "a", ScopedProviderID: "p", HeldTimeSlot: "slot"})
	if !errors.Is(err, errBoom) {
		t.Fatalf("Book error = %v, want errBoom", err)
	}
}

// TestAnalyticsDashboardRepository_RoundTrip covers the dashboard aggregate CRUD
// (Save, FindByID and the doc ID accessor) which the rollup tests do not reach.
func TestAnalyticsDashboardRepository_RoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewAnalyticsDashboardRepository(NewMemStore(), nil)

	d := &admin.AnalyticsDashboardAggregate{ID: "dash-1", ExposesPHI: true, RollupOutOfScope: true}
	if err := repo.Save(ctx, d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.FindByID(ctx, "dash-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID != "dash-1" || !got.ExposesPHI || !got.RollupOutOfScope {
		t.Fatalf("dashboard round trip mismatch: %+v", got)
	}
	if got.Version != 1 {
		t.Fatalf("expected persisted version 1, got %d", got.Version)
	}
}

// TestAnalyticsDashboard_RollupErrors covers the fact-source error branch of each
// rollup query.
func TestAnalyticsDashboard_RollupErrors(t *testing.T) {
	ctx := context.Background()
	repo := NewAnalyticsDashboardRepository(NewMemStore(), errFactSource{errBoom})
	from, to := time.Time{}, time.Time{}

	if _, err := repo.Utilization(ctx, "c1", from, to); !errors.Is(err, errBoom) {
		t.Fatalf("Utilization error = %v, want errBoom", err)
	}
	if _, err := repo.NoShow(ctx, "c1", from, to); !errors.Is(err, errBoom) {
		t.Fatalf("NoShow error = %v, want errBoom", err)
	}
	if _, err := repo.Revenue(ctx, "c1", from, to); !errors.Is(err, errBoom) {
		t.Fatalf("Revenue error = %v, want errBoom", err)
	}
}

// TestClinicDirectoryRepository_Counts_Error covers the error propagation of the
// Counts rollup when the underlying load fails.
func TestClinicDirectoryRepository_Counts_Error(t *testing.T) {
	repo := NewClinicDirectoryRepository(errStore{errBoom})
	if _, err := repo.Counts(context.Background(), "x"); !errors.Is(err, errBoom) {
		t.Fatalf("Counts error = %v, want errBoom", err)
	}
}

// TestDecryptField covers both the never-encrypted short circuit and the
// decryption-failure branch (a ciphertext sealed under a different key).
func TestDecryptField(t *testing.T) {
	ctx := context.Background()

	// A zero-version CipherText decodes to the empty string without touching the
	// cipher.
	if got, err := decryptField(ctx, testCipher(t), crypto.CipherText{}); err != nil || got != "" {
		t.Fatalf("decryptField(zero) = (%q, %v), want (\"\", nil)", got, err)
	}

	sealed, err := encryptField(ctx, testCipher(t), "secret-phi")
	if err != nil {
		t.Fatalf("encryptField: %v", err)
	}
	otherEnv, err := crypto.NewAESKeyEnvelope("other-key", bytes.Repeat([]byte{0x9f}, crypto.KeySize))
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	otherCipher := crypto.NewFieldCipher(otherEnv)
	if _, err := decryptField(ctx, otherCipher, sealed); err == nil {
		t.Fatal("expected a decryption failure under a mismatched key")
	}
}

// TestEncounterRepository_FindByID_DecryptError covers the fromDoc decryption
// error branch: a stored encounter whose PHI was sealed under a different cipher
// cannot be decrypted on load.
func TestEncounterRepository_FindByID_DecryptError(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()

	writer := NewEncounterRepository(store, testCipher(t))
	enc := &clin.EncounterAggregate{
		ID:        "enc-x",
		Status:    clin.EncounterStatus("open"),
		Note:      &clin.ClinicalNote{Content: "acute pain", Signed: true},
		Diagnoses: []clin.Diagnosis{{Code: "I20.9", Description: "angina"}},
	}
	if err := writer.Save(ctx, enc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	otherEnv, err := crypto.NewAESKeyEnvelope("other-key", bytes.Repeat([]byte{0x11}, crypto.KeySize))
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	reader := NewEncounterRepository(store, crypto.NewFieldCipher(otherEnv))
	if _, err := reader.FindByID(ctx, "enc-x"); err == nil {
		t.Fatal("expected a decryption failure reading PHI sealed under a different key")
	}
}
