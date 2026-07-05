package mongodb

import (
	"context"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	adminrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/repository"
)

// analyticsDashboardsCollection is the collection analytics-dashboard documents live in.
const analyticsDashboardsCollection = "analytics_dashboards"

// analyticsDashboardDoc is the at-rest projection of an AnalyticsDashboardAggregate.
type analyticsDashboardDoc struct {
	DocID string `bson:"_id"`
	Ver   int    `bson:"version"`

	RollupOutOfScope      bool `bson:"rollup_out_of_scope"`
	ExposesPHI            bool `bson:"exposes_phi"`
	RollupNotReproducible bool `bson:"rollup_not_reproducible"`
}

func (d *analyticsDashboardDoc) ID() string       { return d.DocID }
func (d *analyticsDashboardDoc) Version() int     { return d.Ver }
func (d *analyticsDashboardDoc) SetVersion(v int) { d.Ver = v }

// UtilizationRollup reports how fully a clinic's offered slots were used over a
// window: filled slots out of total offered, plus the derived rate.
type UtilizationRollup struct {
	TotalSlots  int
	FilledSlots int
	Rate        float64
}

// NoShowRollup reports how many scheduled visits ended as no-shows over a window,
// plus the derived rate.
type NoShowRollup struct {
	ScheduledVisits int
	NoShows         int
	Rate            float64
}

// RevenueRollup reports the total captured revenue over a window, in whole cents,
// and the number of payments that made it up.
type RevenueRollup struct {
	CapturedCents int64
	PaymentCount  int
}

// AnalyticsDashboardRepository is the MongoDB-backed AnalyticsDashboardRepository.
// Beyond persisting the dashboard aggregate it computes the utilization, no-show
// and revenue rollups from a FactSource — the aggregation queries the dashboard
// surfaces are built on.
type AnalyticsDashboardRepository struct {
	base  *BaseRepository
	facts FactSource
}

var _ adminrepo.AnalyticsDashboardRepository = (*AnalyticsDashboardRepository)(nil)

// NewAnalyticsDashboardRepository builds an analytics-dashboard repository over a
// store and the fact source its rollups aggregate. facts may be nil if only the
// aggregate CRUD is needed.
func NewAnalyticsDashboardRepository(store DocumentStore, facts FactSource) *AnalyticsDashboardRepository {
	return &AnalyticsDashboardRepository{base: NewBaseRepository(store, analyticsDashboardsCollection), facts: facts}
}

// Save persists the analytics-dashboard aggregate with optimistic concurrency.
func (r *AnalyticsDashboardRepository) Save(ctx context.Context, a *model.AnalyticsDashboardAggregate) error {
	doc := &analyticsDashboardDoc{
		DocID:                 a.ID,
		Ver:                   a.GetVersion(),
		RollupOutOfScope:      a.RollupOutOfScope,
		ExposesPHI:            a.ExposesPHI,
		RollupNotReproducible: a.RollupNotReproducible,
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads an analytics-dashboard aggregate by identity.
func (r *AnalyticsDashboardRepository) FindByID(ctx context.Context, id string) (*model.AnalyticsDashboardAggregate, error) {
	var doc analyticsDashboardDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	a := &model.AnalyticsDashboardAggregate{
		ID:                    doc.DocID,
		RollupOutOfScope:      doc.RollupOutOfScope,
		ExposesPHI:            doc.ExposesPHI,
		RollupNotReproducible: doc.RollupNotReproducible,
	}
	a.Version = doc.Ver
	return a, nil
}

// Utilization rolls up slot utilization for a clinic over [from, to): the share
// of offered slots that were filled (booked, completed, or attended-but-no-show).
func (r *AnalyticsDashboardRepository) Utilization(ctx context.Context, clinicID string, from, to time.Time) (UtilizationRollup, error) {
	facts, err := r.facts.AppointmentFacts(ctx, clinicID, from, to)
	if err != nil {
		return UtilizationRollup{}, err
	}
	roll := UtilizationRollup{TotalSlots: len(facts)}
	for _, f := range facts {
		if f.Status == FactStatusBooked || f.Status == FactStatusCompleted || f.Status == FactStatusNoShow {
			roll.FilledSlots++
		}
	}
	if roll.TotalSlots > 0 {
		roll.Rate = float64(roll.FilledSlots) / float64(roll.TotalSlots)
	}
	return roll, nil
}

// NoShow rolls up the no-show rate for a clinic over [from, to): no-shows out of
// scheduled visits (booked, completed, or no-show — slots a patient was expected at).
func (r *AnalyticsDashboardRepository) NoShow(ctx context.Context, clinicID string, from, to time.Time) (NoShowRollup, error) {
	facts, err := r.facts.AppointmentFacts(ctx, clinicID, from, to)
	if err != nil {
		return NoShowRollup{}, err
	}
	var roll NoShowRollup
	for _, f := range facts {
		switch f.Status {
		case FactStatusBooked, FactStatusCompleted, FactStatusNoShow:
			roll.ScheduledVisits++
			if f.Status == FactStatusNoShow {
				roll.NoShows++
			}
		}
	}
	if roll.ScheduledVisits > 0 {
		roll.Rate = float64(roll.NoShows) / float64(roll.ScheduledVisits)
	}
	return roll, nil
}

// Revenue rolls up captured revenue for a clinic over [from, to): the summed
// payment amounts and the count of payments.
func (r *AnalyticsDashboardRepository) Revenue(ctx context.Context, clinicID string, from, to time.Time) (RevenueRollup, error) {
	facts, err := r.facts.RevenueFacts(ctx, clinicID, from, to)
	if err != nil {
		return RevenueRollup{}, err
	}
	roll := RevenueRollup{PaymentCount: len(facts)}
	for _, f := range facts {
		roll.CapturedCents += f.AmountCents
	}
	return roll, nil
}
