package mocks

import (
	"context"
	"errors"
	"testing"

	adminmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	authzmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	billingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	clinicalmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	identitymodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/model"
	engagementmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	schedulingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
)

// The mocks in this package come in two flavors that must each be exercised on
// both branches of every method:
//
//   - "nil-flavor" repositories return (nil, nil) from FindByID when the id is
//     absent.
//   - "err-flavor" repositories return (nil, ErrXNotFound) from FindByID when
//     the id is absent.
//
// For each repository the tests below:
//  1. Save an aggregate created via the constructor and read it back (Save
//     happy path + FindByID found branch).
//  2. Look up a missing id (FindByID not-found branch: nil or sentinel error).
//  3. Save into a zero-valued struct literal whose backing map is nil, which
//     exercises the lazy `make(...)` initialization branch inside Save.

func TestInMemoryInvoiceRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryInvoiceRepository()

	agg := &billingmodel.InvoiceAggregate{ID: "inv-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "inv-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	missing, err := repo.FindByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = (%v, %v), want (nil, nil)", missing, err)
	}

	nilMap := &InMemoryInvoiceRepository{}
	if err := nilMap.Save(ctx, &billingmodel.InvoiceAggregate{ID: "inv-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "inv-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryInsurancePolicyRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryInsurancePolicyRepository()

	agg := &billingmodel.InsurancePolicyAggregate{ID: "pol-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "pol-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, ErrInsurancePolicyNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrInsurancePolicyNotFound", err)
	}

	nilMap := &InMemoryInsurancePolicyRepository{}
	if err := nilMap.Save(ctx, &billingmodel.InsurancePolicyAggregate{ID: "pol-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "pol-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryPaymentRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryPaymentRepository()

	agg := &billingmodel.PaymentAggregate{ID: "pay-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "pay-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrPaymentNotFound", err)
	}

	nilMap := &InMemoryPaymentRepository{}
	if err := nilMap.Save(ctx, &billingmodel.PaymentAggregate{ID: "pay-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "pay-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemorySessionRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemorySessionRepository()

	agg := &identitymodel.SessionAggregate{ID: "sess-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "sess-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrSessionNotFound", err)
	}

	nilMap := &InMemorySessionRepository{}
	if err := nilMap.Save(ctx, &identitymodel.SessionAggregate{ID: "sess-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "sess-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryUserAccountRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryUserAccountRepository()

	agg := &identitymodel.UserAccountAggregate{ID: "usr-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "usr-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, ErrUserAccountNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrUserAccountNotFound", err)
	}

	nilMap := &InMemoryUserAccountRepository{}
	if err := nilMap.Save(ctx, &identitymodel.UserAccountAggregate{ID: "usr-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "usr-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryCryptoKeyEnvelopeRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryCryptoKeyEnvelopeRepository()

	agg := &auditmodel.CryptoKeyEnvelopeAggregate{ID: "key-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "key-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, ErrCryptoKeyEnvelopeNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrCryptoKeyEnvelopeNotFound", err)
	}

	nilMap := &InMemoryCryptoKeyEnvelopeRepository{}
	if err := nilMap.Save(ctx, &auditmodel.CryptoKeyEnvelopeAggregate{ID: "key-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "key-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryAuditTrailRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryAuditTrailRepository()

	agg := &auditmodel.AuditTrailAggregate{ID: "aud-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "aud-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, ErrAuditTrailNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrAuditTrailNotFound", err)
	}

	nilMap := &InMemoryAuditTrailRepository{}
	if err := nilMap.Save(ctx, &auditmodel.AuditTrailAggregate{ID: "aud-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "aud-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryMessageThreadRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryMessageThreadRepository()

	agg := &engagementmodel.MessageThreadAggregate{ID: "msg-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "msg-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, ErrMessageThreadNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrMessageThreadNotFound", err)
	}

	nilMap := &InMemoryMessageThreadRepository{}
	if err := nilMap.Save(ctx, &engagementmodel.MessageThreadAggregate{ID: "msg-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "msg-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryProviderScheduleRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryProviderScheduleRepository()

	agg := &schedulingmodel.ProviderScheduleAggregate{ID: "sch-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "sch-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, ErrProviderScheduleNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrProviderScheduleNotFound", err)
	}

	nilMap := &InMemoryProviderScheduleRepository{}
	if err := nilMap.Save(ctx, &schedulingmodel.ProviderScheduleAggregate{ID: "sch-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "sch-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryAppointmentRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryAppointmentRepository()

	agg := &schedulingmodel.AppointmentAggregate{ID: "apt-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "apt-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	missing, err := repo.FindByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = (%v, %v), want (nil, nil)", missing, err)
	}

	nilMap := &InMemoryAppointmentRepository{}
	if err := nilMap.Save(ctx, &schedulingmodel.AppointmentAggregate{ID: "apt-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "apt-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryEncounterRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryEncounterRepository()

	agg := &clinicalmodel.EncounterAggregate{ID: "enc-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "enc-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	missing, err := repo.FindByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = (%v, %v), want (nil, nil)", missing, err)
	}

	nilMap := &InMemoryEncounterRepository{}
	if err := nilMap.Save(ctx, &clinicalmodel.EncounterAggregate{ID: "enc-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "enc-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryLabOrderRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryLabOrderRepository()

	agg := &clinicalmodel.LabOrderAggregate{ID: "lab-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "lab-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	missing, err := repo.FindByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = (%v, %v), want (nil, nil)", missing, err)
	}

	nilMap := &InMemoryLabOrderRepository{}
	if err := nilMap.Save(ctx, &clinicalmodel.LabOrderAggregate{ID: "lab-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "lab-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryAnalyticsDashboardRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryAnalyticsDashboardRepository()

	agg := &adminmodel.AnalyticsDashboardAggregate{ID: "dash-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "dash-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	missing, err := repo.FindByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = (%v, %v), want (nil, nil)", missing, err)
	}

	nilMap := &InMemoryAnalyticsDashboardRepository{}
	if err := nilMap.Save(ctx, &adminmodel.AnalyticsDashboardAggregate{ID: "dash-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "dash-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryClinicDirectoryRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryClinicDirectoryRepository()

	agg := &adminmodel.ClinicDirectoryAggregate{ID: "clinic-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "clinic-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	missing, err := repo.FindByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = (%v, %v), want (nil, nil)", missing, err)
	}

	nilMap := &InMemoryClinicDirectoryRepository{}
	if err := nilMap.Save(ctx, &adminmodel.ClinicDirectoryAggregate{ID: "clinic-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "clinic-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryAuthorizationPolicyRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryAuthorizationPolicyRepository()

	agg := &authzmodel.AuthorizationPolicyAggregate{ID: "authz-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "authz-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	missing, err := repo.FindByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = (%v, %v), want (nil, nil)", missing, err)
	}

	nilMap := &InMemoryAuthorizationPolicyRepository{}
	if err := nilMap.Save(ctx, &authzmodel.AuthorizationPolicyAggregate{ID: "authz-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "authz-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}

func TestInMemoryCareRelationshipRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryCareRelationshipRepository()

	agg := &authzmodel.CareRelationshipAggregate{ID: "care-1"}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := repo.FindByID(ctx, "care-1")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got != agg {
		t.Fatalf("FindByID = %v, want %v", got, agg)
	}

	missing, err := repo.FindByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Fatalf("FindByID(missing) = (%v, %v), want (nil, nil)", missing, err)
	}

	nilMap := &InMemoryCareRelationshipRepository{}
	if err := nilMap.Save(ctx, &authzmodel.CareRelationshipAggregate{ID: "care-2"}); err != nil {
		t.Fatalf("Save on nil map returned error: %v", err)
	}
	if got, _ := nilMap.FindByID(ctx, "care-2"); got == nil {
		t.Fatal("Save on nil map did not persist aggregate")
	}
}
