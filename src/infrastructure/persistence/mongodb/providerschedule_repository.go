package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	schedrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
)

// providerSchedulesCollection is the collection provider-schedule documents live in.
const providerSchedulesCollection = "provider_schedules"

// providerScheduleDoc is the at-rest projection of a ProviderScheduleAggregate.
type providerScheduleDoc struct {
	DocID string `bson:"_id"`
	Ver   int    `bson:"version"`

	ScopedProviderID string   `bson:"scoped_provider_id"`
	PublishedWindows []string `bson:"published_windows"`
	BlockedIntervals []string `bson:"blocked_intervals"`

	WindowsOverlap              bool `bson:"windows_overlap"`
	WindowOffersBlockedInterval bool `bson:"window_offers_blocked_interval"`
	WindowOutsideOperatingHours bool `bson:"window_outside_operating_hours"`
}

func (d *providerScheduleDoc) ID() string       { return d.DocID }
func (d *providerScheduleDoc) Version() int     { return d.Ver }
func (d *providerScheduleDoc) SetVersion(v int) { d.Ver = v }

// ProviderScheduleRepository is the MongoDB-backed ProviderScheduleRepository.
type ProviderScheduleRepository struct {
	base *BaseRepository
}

var _ schedrepo.ProviderScheduleRepository = (*ProviderScheduleRepository)(nil)

// NewProviderScheduleRepository builds a provider-schedule repository over a store.
func NewProviderScheduleRepository(store DocumentStore) *ProviderScheduleRepository {
	return &ProviderScheduleRepository{base: NewBaseRepository(store, providerSchedulesCollection)}
}

// Save persists the provider-schedule aggregate with optimistic concurrency.
func (r *ProviderScheduleRepository) Save(ctx context.Context, a *model.ProviderScheduleAggregate) error {
	return saveAggregate(ctx, r.base, providerScheduleToDoc(a), a)
}

// FindByID loads a provider-schedule aggregate by identity.
func (r *ProviderScheduleRepository) FindByID(ctx context.Context, id string) (*model.ProviderScheduleAggregate, error) {
	var doc providerScheduleDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	return providerScheduleFromDoc(&doc), nil
}

func providerScheduleToDoc(a *model.ProviderScheduleAggregate) *providerScheduleDoc {
	return &providerScheduleDoc{
		DocID:                       a.ID,
		Ver:                         a.GetVersion(),
		ScopedProviderID:            a.ScopedProviderID,
		PublishedWindows:            a.PublishedWindows,
		BlockedIntervals:            a.BlockedIntervals,
		WindowsOverlap:              a.WindowsOverlap,
		WindowOffersBlockedInterval: a.WindowOffersBlockedInterval,
		WindowOutsideOperatingHours: a.WindowOutsideOperatingHours,
	}
}

func providerScheduleFromDoc(d *providerScheduleDoc) *model.ProviderScheduleAggregate {
	a := &model.ProviderScheduleAggregate{
		ID:                          d.DocID,
		ScopedProviderID:            d.ScopedProviderID,
		PublishedWindows:            d.PublishedWindows,
		BlockedIntervals:            d.BlockedIntervals,
		WindowsOverlap:              d.WindowsOverlap,
		WindowOffersBlockedInterval: d.WindowOffersBlockedInterval,
		WindowOutsideOperatingHours: d.WindowOutsideOperatingHours,
	}
	a.Version = d.Ver
	return a
}
