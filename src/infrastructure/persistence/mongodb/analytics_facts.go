package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Appointment slot dispositions the utilization and no-show rollups count over.
// They are the persisted `status` values a scheduling read model exposes for a
// slot in a reporting window.
const (
	FactStatusOpen      = "open"
	FactStatusBooked    = "booked"
	FactStatusCompleted = "completed"
	FactStatusNoShow    = "no_show"
	FactStatusCancelled = "cancelled"
)

// AppointmentFact is the projection of a scheduled slot the utilization and
// no-show rollups draw on: which clinic offered it, its final disposition, and
// when it was scheduled to start (so a rollup can scope by date).
type AppointmentFact struct {
	ClinicID  string    `bson:"clinic_id"`
	Status    string    `bson:"status"`
	SlotStart time.Time `bson:"slot_start"`
}

// RevenueFact is the projection of a captured payment the revenue rollup sums:
// the clinic it was captured for, the amount in whole cents, and when it was
// captured.
type RevenueFact struct {
	ClinicID    string    `bson:"clinic_id"`
	AmountCents int64     `bson:"amount_cents"`
	CapturedAt  time.Time `bson:"captured_at"`
}

// FactSource supplies the scheduling and billing facts the analytics rollups are
// computed from, scoped to a clinic and a [from, to) date window. Isolating it as
// a port lets the rollup arithmetic be exercised hermetically (MemFactSource)
// while production runs it as MongoDB aggregation pipelines (MongoFactSource).
type FactSource interface {
	AppointmentFacts(ctx context.Context, clinicID string, from, to time.Time) ([]AppointmentFact, error)
	RevenueFacts(ctx context.Context, clinicID string, from, to time.Time) ([]RevenueFact, error)
}

// MemFactSource is an in-memory FactSource for tests and local development. It
// applies the same clinic + date-window filtering the Mongo pipelines do.
type MemFactSource struct {
	Appointments []AppointmentFact
	Revenue      []RevenueFact
}

// AppointmentFacts returns the appointment facts within the clinic and window.
func (m *MemFactSource) AppointmentFacts(_ context.Context, clinicID string, from, to time.Time) ([]AppointmentFact, error) {
	var out []AppointmentFact
	for _, f := range m.Appointments {
		if f.ClinicID == clinicID && inWindow(f.SlotStart, from, to) {
			out = append(out, f)
		}
	}
	return out, nil
}

// RevenueFacts returns the revenue facts within the clinic and window.
func (m *MemFactSource) RevenueFacts(_ context.Context, clinicID string, from, to time.Time) ([]RevenueFact, error) {
	var out []RevenueFact
	for _, f := range m.Revenue {
		if f.ClinicID == clinicID && inWindow(f.CapturedAt, from, to) {
			out = append(out, f)
		}
	}
	return out, nil
}

// inWindow reports whether t falls in [from, to). A zero bound is treated as
// unbounded so callers can request an open-ended window.
func inWindow(t, from, to time.Time) bool {
	if !from.IsZero() && t.Before(from) {
		return false
	}
	if !to.IsZero() && !t.Before(to) {
		return false
	}
	return true
}

// MongoFactSource is the production FactSource. It reads the scheduling and
// billing read-model collections with MongoDB aggregation pipelines that match
// the clinic + date window server-side, so only the in-scope facts cross the
// wire to be rolled up.
type MongoFactSource struct {
	appointments *mongo.Collection
	payments     *mongo.Collection
}

// NewMongoFactSource builds a fact source over the appointment and payment
// read-model collections in a database.
func NewMongoFactSource(db *mongo.Database) *MongoFactSource {
	return &MongoFactSource{
		appointments: db.Collection("appointment_facts"),
		payments:     db.Collection("payment_facts"),
	}
}

// AppointmentFacts runs a $match aggregation scoping to the clinic and window.
func (s *MongoFactSource) AppointmentFacts(ctx context.Context, clinicID string, from, to time.Time) ([]AppointmentFact, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"clinic_id":  clinicID,
			"slot_start": dateRange(from, to),
		}}},
	}
	cur, err := s.appointments.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var out []AppointmentFact
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RevenueFacts runs a $match aggregation scoping to the clinic and window.
func (s *MongoFactSource) RevenueFacts(ctx context.Context, clinicID string, from, to time.Time) ([]RevenueFact, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"clinic_id":   clinicID,
			"captured_at": dateRange(from, to),
		}}},
	}
	cur, err := s.payments.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var out []RevenueFact
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// dateRange builds the [from, to) BSON range predicate, omitting a bound that is
// the zero time so the window can be left open on either end.
func dateRange(from, to time.Time) bson.M {
	rng := bson.M{}
	if !from.IsZero() {
		rng["$gte"] = from
	}
	if !to.IsZero() {
		rng["$lt"] = to
	}
	return rng
}
