package shared

import "testing"

// stubEvent is a minimal DomainEvent used to exercise AggregateRoot's event
// buffer without pulling in a concrete aggregate.
type stubEvent struct {
	typ   string
	aggID string
}

func (e stubEvent) Type() string        { return e.typ }
func (e stubEvent) AggregateID() string { return e.aggID }

func TestAggregateRoot_AddAndEvents(t *testing.T) {
	var root AggregateRoot

	if got := root.Events(); got != nil {
		t.Fatalf("Events() on fresh root = %v, want nil", got)
	}

	e1 := stubEvent{typ: "created", aggID: "agg-1"}
	e2 := stubEvent{typ: "updated", aggID: "agg-1"}
	root.AddEvent(e1)
	root.AddEvent(e2)

	events := root.Events()
	if len(events) != 2 {
		t.Fatalf("Events() len = %d, want 2", len(events))
	}
	if events[0].Type() != "created" || events[1].Type() != "updated" {
		t.Fatalf("Events() = %+v, want [created updated] in order", events)
	}
	if events[0].AggregateID() != "agg-1" {
		t.Fatalf("Events()[0].AggregateID() = %q, want %q", events[0].AggregateID(), "agg-1")
	}
}

func TestAggregateRoot_ClearEvents(t *testing.T) {
	var root AggregateRoot
	root.AddEvent(stubEvent{typ: "created", aggID: "agg-1"})
	root.AddEvent(stubEvent{typ: "updated", aggID: "agg-1"})

	root.ClearEvents()

	if got := root.Events(); got != nil {
		t.Fatalf("Events() after ClearEvents() = %v, want nil", got)
	}

	// AddEvent must still work after a clear.
	root.AddEvent(stubEvent{typ: "reopened", aggID: "agg-1"})
	if got := root.Events(); len(got) != 1 || got[0].Type() != "reopened" {
		t.Fatalf("Events() after re-add = %+v, want single reopened event", got)
	}
}

func TestAggregateRoot_GetVersion(t *testing.T) {
	var root AggregateRoot
	if got := root.GetVersion(); got != 0 {
		t.Fatalf("GetVersion() default = %d, want 0", got)
	}

	root.Version = 7
	if got := root.GetVersion(); got != 7 {
		t.Fatalf("GetVersion() = %d, want 7", got)
	}
}
