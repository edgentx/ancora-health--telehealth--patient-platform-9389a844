package mongodb

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	admin "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	bill "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	clin "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	engage "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	sched "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
)

// testCipher builds a self-contained AES envelope cipher for the encryption
// round-trip tests. The key material is fixed and non-secret — it exists only to
// exercise the encrypt/decrypt path hermetically.
func testCipher(t *testing.T) *crypto.FieldCipher {
	t.Helper()
	env, err := crypto.NewAESKeyEnvelope("test-key", make([]byte, crypto.KeySize))
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	return crypto.NewFieldCipher(env)
}

// TestAppointmentRepository_RoundTrip exercises CRUD: a held appointment is
// saved, reloaded, and must match — the load-time version reconstruction and
// document mapping preserved every field.
func TestAppointmentRepository_RoundTrip(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewAppointmentRepository(store, store, locking.NewMemorySlotLocker(), "")

	a := &sched.AppointmentAggregate{ID: "appt-1"}
	if _, err := a.Execute(sched.HoldSlotCmd{ProviderId: "prov-1", TimeSlot: "2026-07-10T09:00", PatientId: "pat-1"}); err != nil {
		t.Fatalf("HoldSlot: %v", err)
	}
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(ctx, "appt-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != sched.AppointmentStatusHeld || got.HeldTimeSlot != "2026-07-10T09:00" || got.ScopedPatientID != "pat-1" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.GetVersion() != 1 {
		t.Fatalf("expected persisted version 1, got %d", got.GetVersion())
	}
}

// TestAppointmentRepository_ConcurrentBooking is the headline acceptance test:
// two concurrent bookings for the same provider/slot must resolve to exactly one
// success and one typed sched.ErrSlotDoubleBooked conflict.
func TestAppointmentRepository_ConcurrentBooking(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewAppointmentRepository(store, store, locking.NewMemorySlotLocker(), "")

	const provider, slot = "prov-1", "2026-07-10T09:00"
	newHeld := func(id string) *sched.AppointmentAggregate {
		return &sched.AppointmentAggregate{
			ID:               id,
			Status:           sched.AppointmentStatusHeld,
			ScopedProviderID: provider,
			HeldTimeSlot:     slot,
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	appts := []*sched.AppointmentAggregate{newHeld("appt-a"), newHeld("appt-b")}
	for i := range appts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = repo.Book(ctx, appts[i])
		}(i)
	}
	wg.Wait()

	var successes, conflicts int
	for _, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, sched.ErrSlotDoubleBooked):
			conflicts++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected exactly 1 success and 1 conflict, got %d successes / %d conflicts", successes, conflicts)
	}
}

// TestAppointmentRepository_BookThenSameSlotConflicts confirms a sequential
// second booking for a held slot also gets the typed conflict.
func TestAppointmentRepository_BookThenSameSlotConflicts(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewAppointmentRepository(store, store, locking.NewMemorySlotLocker(), "")

	first := &sched.AppointmentAggregate{ID: "appt-a", Status: sched.AppointmentStatusHeld, ScopedProviderID: "prov-1", HeldTimeSlot: "slot-x"}
	if err := repo.Book(ctx, first); err != nil {
		t.Fatalf("first Book: %v", err)
	}
	second := &sched.AppointmentAggregate{ID: "appt-b", Status: sched.AppointmentStatusHeld, ScopedProviderID: "prov-1", HeldTimeSlot: "slot-x"}
	if err := repo.Book(ctx, second); !errors.Is(err, sched.ErrSlotDoubleBooked) {
		t.Fatalf("expected ErrSlotDoubleBooked on second booking, got %v", err)
	}
}

// TestEncounterRepository_PHIEncryptedAtRest verifies the SOAP note body is
// stored as ciphertext (the plaintext never appears in the persisted document)
// and still decrypts back to the original on load.
func TestEncounterRepository_PHIEncryptedAtRest(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewEncounterRepository(store, testCipher(t))

	const secret = "patient reports acute chest pain"
	enc := &clin.EncounterAggregate{
		ID:              "enc-1",
		Status:          clin.EncounterStatus("open"),
		ScopedPatientID: "pat-1",
		Note:            &clin.ClinicalNote{Content: secret, Signed: true},
		Diagnoses:       []clin.Diagnosis{{Code: "I20.9", Description: "angina"}},
	}
	if err := repo.Save(ctx, enc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The stored bytes must not contain the plaintext PHI.
	raw := store.docs["enc-1"]
	if bytes.Contains(raw, []byte(secret)) {
		t.Fatal("plaintext SOAP note leaked into stored document")
	}
	if bytes.Contains(raw, []byte("angina")) {
		t.Fatal("plaintext diagnosis description leaked into stored document")
	}

	got, err := repo.FindByID(ctx, "enc-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Note == nil || got.Note.Content != secret || !got.Note.Signed {
		t.Fatalf("note did not round-trip: %+v", got.Note)
	}
	if len(got.Diagnoses) != 1 || got.Diagnoses[0].Description != "angina" || got.Diagnoses[0].Code != "I20.9" {
		t.Fatalf("diagnosis did not round-trip: %+v", got.Diagnoses)
	}
}

// TestPaymentRepository_PCITokenEncrypted verifies the gateway token is stored
// encrypted and round-trips on load.
func TestPaymentRepository_PCITokenEncrypted(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewPaymentRepository(store, testCipher(t))

	const token = "tok_live_abc123secret"
	pay := &bill.PaymentAggregate{ID: "pay-1", Status: bill.PaymentStatus("captured"), InvoiceID: "inv-1", PaymentToken: token, AmountCents: 5000}
	if err := repo.Save(ctx, pay); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if bytes.Contains(store.docs["pay-1"], []byte(token)) {
		t.Fatal("plaintext gateway token leaked into stored document")
	}
	got, err := repo.FindByID(ctx, "pay-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.PaymentToken != token || got.AmountCents != 5000 {
		t.Fatalf("payment did not round-trip: %+v", got)
	}
}

// TestEncryptedRepositories_RoundTrip covers the remaining encrypting
// repositories (prescription, lab order, insurance policy, intake form): each
// stores its sensitive field as ciphertext and recovers it on load.
func TestEncryptedRepositories_RoundTrip(t *testing.T) {
	ctx := context.Background()
	cipher := testCipher(t)

	t.Run("prescription", func(t *testing.T) {
		store := NewMemStore()
		repo := NewPrescriptionRepository(store, cipher)
		rx := &engage.PrescriptionAggregate{ID: "rx-1", Status: engage.PrescriptionStatus("draft"), Medication: "amoxicillin", Dosage: "500mg"}
		if err := repo.Save(ctx, rx); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if bytes.Contains(store.docs["rx-1"], []byte("amoxicillin")) {
			t.Fatal("plaintext medication leaked")
		}
		got, err := repo.FindByID(ctx, "rx-1")
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.Medication != "amoxicillin" || got.Dosage != "500mg" {
			t.Fatalf("prescription mismatch: %+v", got)
		}
	})

	t.Run("laborder", func(t *testing.T) {
		store := NewMemStore()
		repo := NewLabOrderRepository(store, cipher)
		lo := &clin.LabOrderAggregate{ID: "lo-1", Status: clin.LabOrderStatus("placed"), ScopedPatientID: "pat-1", TestCode: "CBC-2024"}
		if err := repo.Save(ctx, lo); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if bytes.Contains(store.docs["lo-1"], []byte("CBC-2024")) {
			t.Fatal("plaintext test code leaked")
		}
		got, err := repo.FindByID(ctx, "lo-1")
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.TestCode != "CBC-2024" {
			t.Fatalf("lab order mismatch: %+v", got)
		}
	})

	t.Run("insurancepolicy", func(t *testing.T) {
		store := NewMemStore()
		repo := NewInsurancePolicyRepository(store, cipher)
		pol := &bill.InsurancePolicyAggregate{ID: "pol-1", Status: bill.PolicyStatus("active"), PatientID: "pat-1", PayerIdentifier: "AETNA-9911", EffectiveDates: bill.EffectiveDates{Start: "2026-01-01", End: "2026-12-31"}}
		if err := repo.Save(ctx, pol); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if bytes.Contains(store.docs["pol-1"], []byte("AETNA-9911")) {
			t.Fatal("plaintext payer identifier leaked")
		}
		got, err := repo.FindByID(ctx, "pol-1")
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.PayerIdentifier != "AETNA-9911" || got.EffectiveDates.End != "2026-12-31" {
			t.Fatalf("policy mismatch: %+v", got)
		}
	})

	t.Run("intakeform", func(t *testing.T) {
		store := NewMemStore()
		repo := NewIntakeFormRepository(store, cipher)
		form := &engage.IntakeFormAggregate{ID: "if-1", Status: engage.IntakeFormStatus("submitted"), ScopedPatientID: "pat-1", History: "no known allergies", Demographics: "dob 1990"}
		if err := repo.Save(ctx, form); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if bytes.Contains(store.docs["if-1"], []byte("no known allergies")) {
			t.Fatal("plaintext history leaked")
		}
		got, err := repo.FindByID(ctx, "if-1")
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.History != "no known allergies" || got.Demographics != "dob 1990" {
			t.Fatalf("intake form mismatch: %+v", got)
		}
	})
}

// TestPlainRepositories_RoundTrip covers the non-encrypting repositories'
// document mapping (message thread, invoice, provider schedule, clinic directory).
func TestPlainRepositories_RoundTrip(t *testing.T) {
	ctx := context.Background()

	t.Run("providerschedule", func(t *testing.T) {
		store := NewMemStore()
		repo := NewProviderScheduleRepository(store)
		ps := &sched.ProviderScheduleAggregate{ID: "ps-1", ScopedProviderID: "prov-1", PublishedWindows: []string{"mon-am"}, BlockedIntervals: []string{"noon"}}
		if err := repo.Save(ctx, ps); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := repo.FindByID(ctx, "ps-1")
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if len(got.PublishedWindows) != 1 || got.PublishedWindows[0] != "mon-am" || got.BlockedIntervals[0] != "noon" {
			t.Fatalf("schedule mismatch: %+v", got)
		}
	})

	t.Run("invoice", func(t *testing.T) {
		store := NewMemStore()
		repo := NewInvoiceRepository(store)
		inv := &bill.InvoiceAggregate{ID: "inv-1", Status: bill.InvoiceStatus("open"), EncounterID: "enc-1", LineItems: []bill.InvoiceLineItem{{Description: "office visit", AmountCents: 12000}}, CoverageCents: 8000, CopayCents: 2000}
		if err := repo.Save(ctx, inv); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := repo.FindByID(ctx, "inv-1")
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if len(got.LineItems) != 1 || got.LineItems[0].AmountCents != 12000 || got.CoverageCents != 8000 {
			t.Fatalf("invoice mismatch: %+v", got)
		}
	})

	t.Run("messagethread", func(t *testing.T) {
		store := NewMemStore()
		repo := NewMessageThreadRepository(store)
		mt := &engage.MessageThreadAggregate{ID: "mt-1", Status: engage.MessageThreadStatus("open"), ScopedPatientID: "pat-1", Subject: "follow-up", PostedMessageCount: 3}
		if err := repo.Save(ctx, mt); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := repo.FindByID(ctx, "mt-1")
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.Subject != "follow-up" || got.PostedMessageCount != 3 {
			t.Fatalf("thread mismatch: %+v", got)
		}
	})

	t.Run("clinicdirectory", func(t *testing.T) {
		store := NewMemStore()
		repo := NewClinicDirectoryRepository(store)
		cd := &admin.ClinicDirectoryAggregate{ID: "cd-1", ProviderIDs: []string{"p1", "p2"}, SpecialtyCodes: []string{"CARD"}, ClinicIDs: []string{"c1"}}
		if err := repo.Save(ctx, cd); err != nil {
			t.Fatalf("Save: %v", err)
		}
		counts, err := repo.Counts(ctx, "cd-1")
		if err != nil {
			t.Fatalf("Counts: %v", err)
		}
		if counts.Providers != 2 || counts.Specialties != 1 || counts.Clinics != 1 {
			t.Fatalf("directory counts mismatch: %+v", counts)
		}
	})
}

// TestAnalyticsDashboard_Rollups verifies query correctness for the utilization,
// no-show and revenue aggregations against a deterministic fact set.
func TestAnalyticsDashboard_Rollups(t *testing.T) {
	ctx := context.Background()
	day := func(d int) time.Time { return time.Date(2026, 7, d, 9, 0, 0, 0, time.UTC) }

	facts := &MemFactSource{
		Appointments: []AppointmentFact{
			{ClinicID: "c1", Status: FactStatusBooked, SlotStart: day(1)},
			{ClinicID: "c1", Status: FactStatusCompleted, SlotStart: day(2)},
			{ClinicID: "c1", Status: FactStatusNoShow, SlotStart: day(3)},
			{ClinicID: "c1", Status: FactStatusOpen, SlotStart: day(4)},
			{ClinicID: "c2", Status: FactStatusBooked, SlotStart: day(1)}, // other clinic — excluded
		},
		Revenue: []RevenueFact{
			{ClinicID: "c1", AmountCents: 10000, CapturedAt: day(1)},
			{ClinicID: "c1", AmountCents: 5000, CapturedAt: day(2)},
			{ClinicID: "c2", AmountCents: 9999, CapturedAt: day(1)}, // other clinic — excluded
		},
	}
	repo := NewAnalyticsDashboardRepository(NewMemStore(), facts)

	from, to := day(1), day(10)

	util, err := repo.Utilization(ctx, "c1", from, to)
	if err != nil {
		t.Fatalf("Utilization: %v", err)
	}
	// 4 slots for c1, 3 filled (booked, completed, no-show).
	if util.TotalSlots != 4 || util.FilledSlots != 3 {
		t.Fatalf("utilization counts: %+v", util)
	}
	if util.Rate < 0.749 || util.Rate > 0.751 {
		t.Fatalf("utilization rate: %v", util.Rate)
	}

	ns, err := repo.NoShow(ctx, "c1", from, to)
	if err != nil {
		t.Fatalf("NoShow: %v", err)
	}
	// 3 scheduled visits, 1 no-show.
	if ns.ScheduledVisits != 3 || ns.NoShows != 1 {
		t.Fatalf("no-show counts: %+v", ns)
	}

	rev, err := repo.Revenue(ctx, "c1", from, to)
	if err != nil {
		t.Fatalf("Revenue: %v", err)
	}
	if rev.CapturedCents != 15000 || rev.PaymentCount != 2 {
		t.Fatalf("revenue rollup: %+v", rev)
	}
}

// TestAnalyticsDashboard_WindowScoping confirms the date window bounds the
// rollup: facts outside [from, to) are excluded.
func TestAnalyticsDashboard_WindowScoping(t *testing.T) {
	ctx := context.Background()
	day := func(d int) time.Time { return time.Date(2026, 7, d, 9, 0, 0, 0, time.UTC) }
	facts := &MemFactSource{
		Revenue: []RevenueFact{
			{ClinicID: "c1", AmountCents: 100, CapturedAt: day(1)},
			{ClinicID: "c1", AmountCents: 200, CapturedAt: day(20)}, // outside window
		},
	}
	repo := NewAnalyticsDashboardRepository(NewMemStore(), facts)
	rev, err := repo.Revenue(ctx, "c1", day(1), day(10))
	if err != nil {
		t.Fatalf("Revenue: %v", err)
	}
	if rev.CapturedCents != 100 || rev.PaymentCount != 1 {
		t.Fatalf("window scoping failed: %+v", rev)
	}
}
