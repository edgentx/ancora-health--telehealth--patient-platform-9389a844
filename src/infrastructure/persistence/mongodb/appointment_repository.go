package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	schedrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
)

// appointmentsCollection is the collection appointment documents live in.
const appointmentsCollection = "appointments"

// appointmentDoc is the at-rest projection of an AppointmentAggregate. Every
// field the aggregate exposes is mapped explicitly so a stored document round
// trips back into an identical aggregate. The version field carries the
// optimistic-concurrency counter.
type appointmentDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	ScopedProviderID string `bson:"scoped_provider_id"`
	ScopedPatientID  string `bson:"scoped_patient_id"`
	HeldTimeSlot     string `bson:"held_time_slot"`

	SlotOutsideAvailability bool `bson:"slot_outside_availability"`
	SlotAlreadyBooked       bool `bson:"slot_already_booked"`
	HoldLockExpired         bool `bson:"hold_lock_expired"`
	OutsidePolicyWindow     bool `bson:"outside_policy_window"`
}

func (d *appointmentDoc) ID() string       { return d.DocID }
func (d *appointmentDoc) Version() int     { return d.Ver }
func (d *appointmentDoc) SetVersion(v int) { d.Ver = v }

// AppointmentRepository is the MongoDB-backed AppointmentRepository. Beyond CRUD
// it owns the slot-hold lock and transaction runner that make booking safe under
// concurrency: a booking acquires an exclusive slot lock, then commits the
// aggregate inside a MongoDB transaction, so two racing bookings for the same
// slot resolve to exactly one success and one typed conflict.
type AppointmentRepository struct {
	base      *BaseRepository
	tx        TransactionRunner
	locker    locking.SlotLocker
	projectID string
}

// compile-time assertion that the concrete type satisfies the domain port.
var _ schedrepo.AppointmentRepository = (*AppointmentRepository)(nil)

// NewAppointmentRepository builds an appointment repository over a store, a
// transaction runner and a slot locker. An empty projectID falls back to the
// project default so slot keys are always namespaced.
func NewAppointmentRepository(store DocumentStore, tx TransactionRunner, locker locking.SlotLocker, projectID string) *AppointmentRepository {
	if projectID == "" {
		projectID = locking.DefaultProjectID
	}
	return &AppointmentRepository{
		base:      NewBaseRepository(store, appointmentsCollection),
		tx:        tx,
		locker:    locker,
		projectID: projectID,
	}
}

// Save persists the appointment aggregate with optimistic-concurrency control.
// It is the plain write path; Book adds the slot-lock + transaction guarantees
// required to confirm a booking.
func (r *AppointmentRepository) Save(ctx context.Context, a *model.AppointmentAggregate) error {
	return saveAggregate(ctx, r.base, appointmentToDoc(a), a)
}

// Book confirms an appointment onto its held slot under an exclusive slot-hold
// lock, committing the aggregate inside a MongoDB transaction. It acquires the
// project-namespaced lock for the appointment's provider/time-slot pair; if the
// slot is already held by another booking it returns model.ErrSlotDoubleBooked
// — the domain's typed double-booking conflict — without touching the database.
// On a persistence failure the hold is released so the slot is not wedged; on
// success the hold is retained (with its TTL) to keep the confirmed slot
// reserved.
func (r *AppointmentRepository) Book(ctx context.Context, a *model.AppointmentAggregate) error {
	key := locking.SlotKey(r.projectID, a.ScopedProviderID, a.HeldTimeSlot)

	acquired, err := r.locker.Acquire(ctx, key, a.ID, locking.DefaultHoldTTL)
	if err != nil {
		// A held slot is the one non-fatal outcome: surface it as the domain
		// double-booking conflict rather than an infrastructure error.
		if err == locking.ErrSlotHeld {
			return model.ErrSlotDoubleBooked
		}
		return err
	}
	if !acquired {
		return model.ErrSlotDoubleBooked
	}

	if err := r.tx.RunInTransaction(ctx, func(txCtx context.Context) error {
		return r.Save(txCtx, a)
	}); err != nil {
		_ = r.locker.Release(ctx, key, a.ID)
		return err
	}
	return nil
}

// FindByID loads an appointment aggregate by identity, returning
// ErrDocumentNotFound when it does not exist.
func (r *AppointmentRepository) FindByID(ctx context.Context, id string) (*model.AppointmentAggregate, error) {
	var doc appointmentDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	return appointmentFromDoc(&doc), nil
}

// appointmentToDoc maps an aggregate onto its persistence document.
func appointmentToDoc(a *model.AppointmentAggregate) *appointmentDoc {
	return &appointmentDoc{
		DocID:                   a.ID,
		Ver:                     a.GetVersion(),
		StatusV:                 string(a.Status),
		ScopedProviderID:        a.ScopedProviderID,
		ScopedPatientID:         a.ScopedPatientID,
		HeldTimeSlot:            a.HeldTimeSlot,
		SlotOutsideAvailability: a.SlotOutsideAvailability,
		SlotAlreadyBooked:       a.SlotAlreadyBooked,
		HoldLockExpired:         a.HoldLockExpired,
		OutsidePolicyWindow:     a.OutsidePolicyWindow,
	}
}

// appointmentFromDoc reconstructs an aggregate from its persistence document.
func appointmentFromDoc(d *appointmentDoc) *model.AppointmentAggregate {
	a := &model.AppointmentAggregate{
		ID:                      d.DocID,
		Status:                  model.AppointmentStatus(d.StatusV),
		ScopedProviderID:        d.ScopedProviderID,
		ScopedPatientID:         d.ScopedPatientID,
		HeldTimeSlot:            d.HeldTimeSlot,
		SlotOutsideAvailability: d.SlotOutsideAvailability,
		SlotAlreadyBooked:       d.SlotAlreadyBooked,
		HoldLockExpired:         d.HoldLockExpired,
		OutsidePolicyWindow:     d.OutsidePolicyWindow,
	}
	a.Version = d.Ver
	return a
}
