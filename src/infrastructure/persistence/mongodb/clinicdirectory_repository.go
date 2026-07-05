package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	adminrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/repository"
)

// clinicDirectoriesCollection is the collection clinic-directory documents live in.
const clinicDirectoriesCollection = "clinic_directories"

// clinicDirectoryDoc is the at-rest projection of a ClinicDirectoryAggregate.
type clinicDirectoryDoc struct {
	DocID string `bson:"_id"`
	Ver   int    `bson:"version"`

	ProviderIDs    []string `bson:"provider_ids"`
	SpecialtyCodes []string `bson:"specialty_codes"`
	ClinicIDs      []string `bson:"clinic_ids"`

	ProviderNotBookable       bool `bson:"provider_not_bookable"`
	DuplicateSpecialtyCode    bool `bson:"duplicate_specialty_code"`
	ClinicDeactivationBlocked bool `bson:"clinic_deactivation_blocked"`
}

func (d *clinicDirectoryDoc) ID() string       { return d.DocID }
func (d *clinicDirectoryDoc) Version() int     { return d.Ver }
func (d *clinicDirectoryDoc) SetVersion(v int) { d.Ver = v }

// DirectoryCounts is the aggregate rollup of a clinic directory: how many
// providers, specialties and clinics it holds. It backs the directory-summary
// query the administration surfaces use.
type DirectoryCounts struct {
	Providers   int
	Specialties int
	Clinics     int
}

// ClinicDirectoryRepository is the MongoDB-backed ClinicDirectoryRepository.
type ClinicDirectoryRepository struct {
	base *BaseRepository
}

var _ adminrepo.ClinicDirectoryRepository = (*ClinicDirectoryRepository)(nil)

// NewClinicDirectoryRepository builds a clinic-directory repository over a store.
func NewClinicDirectoryRepository(store DocumentStore) *ClinicDirectoryRepository {
	return &ClinicDirectoryRepository{base: NewBaseRepository(store, clinicDirectoriesCollection)}
}

// Save persists the clinic-directory aggregate with optimistic concurrency.
func (r *ClinicDirectoryRepository) Save(ctx context.Context, a *model.ClinicDirectoryAggregate) error {
	doc := &clinicDirectoryDoc{
		DocID:                     a.ID,
		Ver:                       a.GetVersion(),
		ProviderIDs:               a.ProviderIDs,
		SpecialtyCodes:            a.SpecialtyCodes,
		ClinicIDs:                 a.ClinicIDs,
		ProviderNotBookable:       a.ProviderNotBookable,
		DuplicateSpecialtyCode:    a.DuplicateSpecialtyCode,
		ClinicDeactivationBlocked: a.ClinicDeactivationBlocked,
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads a clinic-directory aggregate by identity.
func (r *ClinicDirectoryRepository) FindByID(ctx context.Context, id string) (*model.ClinicDirectoryAggregate, error) {
	var doc clinicDirectoryDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	a := &model.ClinicDirectoryAggregate{
		ID:                        doc.DocID,
		ProviderIDs:               doc.ProviderIDs,
		SpecialtyCodes:            doc.SpecialtyCodes,
		ClinicIDs:                 doc.ClinicIDs,
		ProviderNotBookable:       doc.ProviderNotBookable,
		DuplicateSpecialtyCode:    doc.DuplicateSpecialtyCode,
		ClinicDeactivationBlocked: doc.ClinicDeactivationBlocked,
	}
	a.Version = doc.Ver
	return a, nil
}

// Counts returns the directory-summary rollup for a directory: the number of
// registered providers, specialties and clinics. It is the aggregation query the
// administration dashboard draws its directory totals from.
func (r *ClinicDirectoryRepository) Counts(ctx context.Context, id string) (DirectoryCounts, error) {
	a, err := r.FindByID(ctx, id)
	if err != nil {
		return DirectoryCounts{}, err
	}
	return DirectoryCounts{
		Providers:   len(a.ProviderIDs),
		Specialties: len(a.SpecialtyCodes),
		Clinics:     len(a.ClinicIDs),
	}, nil
}
