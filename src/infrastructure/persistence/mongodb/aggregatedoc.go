package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// eventful is the slice of an aggregate the persistence layer needs to reconcile
// the aggregate's self-incrementing version with the store's optimistic-
// concurrency guard. Every domain aggregate satisfies it through the embedded
// shared.AggregateRoot, so repositories accept concrete aggregates and rely on
// this interface only internally.
type eventful interface {
	// GetVersion is the aggregate's current version — advanced once per applied
	// command.
	GetVersion() int
	// Events are the commands' uncommitted events, one per version increment since
	// the aggregate was loaded.
	Events() []shared.DomainEvent
	// ClearEvents drops the uncommitted events once they have been persisted.
	ClearEvents()
}

// loadedVersion recovers the version the aggregate carried when it was last
// loaded from (or first created in) the store. Because every command increments
// the version exactly once and buffers exactly one event, the count of
// uncommitted events is the number of increments since load — so subtracting it
// from the current version yields the load-time version, which is the value the
// store's version guard must match on update.
func loadedVersion(a eventful) int {
	lv := a.GetVersion() - len(a.Events())
	if lv < 0 {
		return 0
	}
	return lv
}

// saveAggregate persists a mapped aggregate document with optimistic-concurrency
// control derived from the aggregate itself. A load-time version of zero means
// the aggregate has never been stored, so it is inserted (stamped version 1);
// otherwise it is updated under a version guard equal to the load-time version,
// turning a lost race into an *OptimisticConcurrencyError. On success the
// aggregate's uncommitted events are cleared, marking them durable.
func saveAggregate(ctx context.Context, base *BaseRepository, doc VersionedDocument, agg eventful) error {
	lv := loadedVersion(agg)
	// Version() must report the load-time version so BaseRepository.Update guards
	// on it; Insert overwrites it with 1.
	doc.SetVersion(lv)

	var err error
	if lv == 0 {
		err = base.Insert(ctx, doc)
	} else {
		err = base.Update(ctx, doc)
	}
	if err != nil {
		return err
	}
	agg.ClearEvents()
	return nil
}
