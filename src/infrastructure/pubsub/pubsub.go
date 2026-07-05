// Package pubsub defines the fan-out port the realtime gateways use to deliver
// messages consistently across horizontally-scaled replicas, plus an in-process
// adapter. A message posted on one replica must reach subscribers connected to
// any other replica, so the gateway publishes to a channel and every replica
// subscribed to that channel receives it — the classic Redis pub/sub pattern.
//
// The port is intentionally narrow (Publish + Subscribe) so the platform can
// wire any concrete broker — go-redis, rueidis, a cluster proxy — without this
// package depending on a specific driver, the same approach the locking package
// takes with its RedisConn port.
package pubsub

import (
	"context"
	"sync"
)

// Message is a single payload delivered on a channel.
type Message struct {
	// Channel is the topic the payload was published to.
	Channel string
	// Payload is the opaque message body. The realtime gateways carry JSON here.
	Payload []byte
}

// Subscription is a live subscription to a channel. Messages published to the
// channel arrive on C until the subscription is closed. Closing releases the
// broker resources backing the subscription and stops delivery.
type Subscription interface {
	// C is the stream of messages delivered to this subscription.
	C() <-chan Message
	// Close ends the subscription and stops delivery.
	Close() error
}

// Broker is the pub/sub fan-out port. Publish broadcasts a payload to every
// subscriber of a channel across all replicas; Subscribe registers interest in a
// channel and returns the live stream.
type Broker interface {
	Publish(ctx context.Context, channel string, payload []byte) error
	Subscribe(ctx context.Context, channel string) (Subscription, error)
}

// MemoryBroker is an in-process Broker. A single instance shared by several
// gateway objects models a shared Redis: a Publish on any gateway fans out to
// the subscriptions held by every other gateway pointed at the same broker,
// which is exactly what lets tests simulate delivery across replicas without a
// live Redis.
type MemoryBroker struct {
	// buffer bounds each subscription's delivery channel so a slow consumer
	// cannot block a publisher indefinitely.
	buffer int

	mu   sync.Mutex
	subs map[string]map[*memSubscription]struct{}
}

// NewMemoryBroker builds an empty in-process broker. A non-positive buffer falls
// back to a small default so publishes never block on a momentarily busy
// consumer.
func NewMemoryBroker(buffer int) *MemoryBroker {
	if buffer <= 0 {
		buffer = 16
	}
	return &MemoryBroker{
		buffer: buffer,
		subs:   make(map[string]map[*memSubscription]struct{}),
	}
}

// Publish delivers payload to every live subscription on channel. Delivery is
// best-effort per subscriber: a subscriber whose buffer is full is skipped
// rather than blocking the publisher, mirroring Redis pub/sub's fire-and-forget
// semantics where a wedged consumer never stalls the wider fan-out.
func (b *MemoryBroker) Publish(_ context.Context, channel string, payload []byte) error {
	b.mu.Lock()
	targets := make([]*memSubscription, 0, len(b.subs[channel]))
	for s := range b.subs[channel] {
		targets = append(targets, s)
	}
	b.mu.Unlock()

	for _, s := range targets {
		// Copy the payload per subscriber so concurrent readers never share the
		// caller's backing array.
		cp := make([]byte, len(payload))
		copy(cp, payload)
		select {
		case s.ch <- Message{Channel: channel, Payload: cp}:
		default:
		}
	}
	return nil
}

// Subscribe registers a new subscription on channel.
func (b *MemoryBroker) Subscribe(_ context.Context, channel string) (Subscription, error) {
	sub := &memSubscription{
		broker:  b,
		channel: channel,
		ch:      make(chan Message, b.buffer),
	}
	b.mu.Lock()
	if b.subs[channel] == nil {
		b.subs[channel] = make(map[*memSubscription]struct{})
	}
	b.subs[channel][sub] = struct{}{}
	b.mu.Unlock()
	return sub, nil
}

// memSubscription is a MemoryBroker subscription.
type memSubscription struct {
	broker  *MemoryBroker
	channel string
	ch      chan Message

	once sync.Once
}

func (s *memSubscription) C() <-chan Message { return s.ch }

// Close deregisters the subscription and closes its delivery channel. It is safe
// to call more than once.
func (s *memSubscription) Close() error {
	s.once.Do(func() {
		s.broker.mu.Lock()
		if set := s.broker.subs[s.channel]; set != nil {
			delete(set, s)
			if len(set) == 0 {
				delete(s.broker.subs, s.channel)
			}
		}
		s.broker.mu.Unlock()
		close(s.ch)
	})
	return nil
}

// Compile-time assertions that the in-process types satisfy their ports.
var (
	_ Broker       = (*MemoryBroker)(nil)
	_ Subscription = (*memSubscription)(nil)
)
