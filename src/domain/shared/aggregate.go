package shared

// Aggregate is the contract every domain aggregate must satisfy.
//
// Execute applies a command to the aggregate and returns the domain events it
// produced (or an error). ID exposes the aggregate's identity, and GetVersion
// reports the current version used for optimistic-concurrency checks.
type Aggregate interface {
	Execute(cmd interface{}) ([]DomainEvent, error)
	ID() string
	GetVersion() int
}

// AggregateRoot is the embeddable base that provides version tracking and an
// in-memory buffer of uncommitted domain events. Concrete aggregates embed it
// and supply their own ID() and Execute() implementations.
type AggregateRoot struct {
	Version int

	events []DomainEvent
}

// AddEvent appends a domain event to the uncommitted buffer.
func (a *AggregateRoot) AddEvent(event DomainEvent) {
	a.events = append(a.events, event)
}

// Events returns the uncommitted domain events buffered on the aggregate.
func (a *AggregateRoot) Events() []DomainEvent {
	return a.events
}

// ClearEvents drops every uncommitted event, typically after they are persisted.
func (a *AggregateRoot) ClearEvents() {
	a.events = nil
}

// GetVersion returns the aggregate's current version for optimistic-concurrency
// control.
func (a *AggregateRoot) GetVersion() int {
	return a.Version
}
