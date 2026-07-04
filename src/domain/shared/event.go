package shared

// DomainEvent is implemented by every event emitted from an aggregate. Type
// identifies the kind of event; AggregateID ties the event back to the
// aggregate instance that produced it.
type DomainEvent interface {
	Type() string
	AggregateID() string
}
