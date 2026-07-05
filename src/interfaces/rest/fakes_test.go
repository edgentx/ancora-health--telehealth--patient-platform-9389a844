package rest

import (
	"context"

	adminmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	adminrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/repository"
	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	auditrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/repository"
	billingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
	clinicalmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	clinicalrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/repository"
	engagementmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	engagementrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	schedulingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// fakeRepo is an in-memory implementation of the uniform Save/FindByID port every
// aggregate's repository exposes. It lets the handler suite drive success,
// not-found and conflict paths without any infrastructure: FindByID returns the
// same mongodb.ErrDocumentNotFound the real repositories emit, and saveErr forces
// a persistence failure (e.g. an optimistic-concurrency conflict) on demand.
type fakeRepo[T any] struct {
	items   map[string]*T
	idOf    func(*T) string
	saveErr error
}

func newFakeRepo[T any](idOf func(*T) string) *fakeRepo[T] {
	return &fakeRepo[T]{items: make(map[string]*T), idOf: idOf}
}

func (f *fakeRepo[T]) Save(_ context.Context, a *T) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.items[f.idOf(a)] = a
	return nil
}

func (f *fakeRepo[T]) FindByID(_ context.Context, id string) (*T, error) {
	if a, ok := f.items[id]; ok {
		return a, nil
	}
	return nil, mongodb.ErrDocumentNotFound
}

// seed inserts an aggregate directly, bypassing Save, so a test can arrange a
// pre-existing record (including one in a state that triggers a domain conflict).
func (f *fakeRepo[T]) seed(a *T) { f.items[f.idOf(a)] = a }

// fakes bundles one fake per exposed aggregate so a test can seed records and
// force save errors after building the router.
type fakes struct {
	appts  *fakeRepo[schedmodel.AppointmentAggregate]
	scheds *fakeRepo[schedmodel.ProviderScheduleAggregate]
	labs   *fakeRepo[clinicalmodel.LabOrderAggregate]
	rx     *fakeRepo[engagementmodel.PrescriptionAggregate]
	pols   *fakeRepo[billingmodel.InsurancePolicyAggregate]
	dirs   *fakeRepo[adminmodel.ClinicDirectoryAggregate]
	trails *fakeRepo[auditmodel.AuditTrailAggregate]
}

// newFakes constructs the fake set with each aggregate's ID accessor.
func newFakes() fakes {
	return fakes{
		appts:  newFakeRepo(func(a *schedmodel.AppointmentAggregate) string { return a.ID }),
		scheds: newFakeRepo(func(a *schedmodel.ProviderScheduleAggregate) string { return a.ID }),
		labs:   newFakeRepo(func(a *clinicalmodel.LabOrderAggregate) string { return a.ID }),
		rx:     newFakeRepo(func(a *engagementmodel.PrescriptionAggregate) string { return a.ID }),
		pols:   newFakeRepo(func(a *billingmodel.InsurancePolicyAggregate) string { return a.ID }),
		dirs:   newFakeRepo(func(a *adminmodel.ClinicDirectoryAggregate) string { return a.ID }),
		trails: newFakeRepo(func(a *auditmodel.AuditTrailAggregate) string { return a.ID }),
	}
}

// deps maps the fakes onto the router Dependencies.
func (f fakes) deps() Dependencies {
	return Dependencies{
		Appointments:      f.appts,
		ProviderSchedules: f.scheds,
		LabOrders:         f.labs,
		Prescriptions:     f.rx,
		InsurancePolicies: f.pols,
		ClinicDirectories: f.dirs,
		AuditTrails:       f.trails,
	}
}

// Compile-time assertions that the generic fake satisfies every port it is
// wired into.
var (
	_ schedulingrepo.AppointmentRepository      = (*fakeRepo[schedmodel.AppointmentAggregate])(nil)
	_ schedulingrepo.ProviderScheduleRepository = (*fakeRepo[schedmodel.ProviderScheduleAggregate])(nil)
	_ clinicalrepo.LabOrderRepository           = (*fakeRepo[clinicalmodel.LabOrderAggregate])(nil)
	_ engagementrepo.PrescriptionRepository     = (*fakeRepo[engagementmodel.PrescriptionAggregate])(nil)
	_ billingrepo.InsurancePolicyRepository     = (*fakeRepo[billingmodel.InsurancePolicyAggregate])(nil)
	_ adminrepo.ClinicDirectoryRepository       = (*fakeRepo[adminmodel.ClinicDirectoryAggregate])(nil)
	_ auditrepo.AuditTrailRepository            = (*fakeRepo[auditmodel.AuditTrailAggregate])(nil)
)
