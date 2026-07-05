package pubsub

import "context"

// RedisClient is the narrow slice of a Redis client the RedisBroker depends on.
// Isolating just Publish and Subscribe keeps the fan-out semantics in one place
// and lets the platform wire any concrete driver (go-redis, rueidis, a cluster
// proxy) without this package taking a compile-time dependency on it — the same
// approach the locking package takes with its RedisConn port.
type RedisClient interface {
	// Publish broadcasts payload to every subscriber of channel across the
	// Redis-connected fleet.
	Publish(ctx context.Context, channel string, payload []byte) error
	// Subscribe opens a subscription to channel. The returned RedisSubscription
	// streams payloads until closed.
	Subscribe(ctx context.Context, channel string) (RedisSubscription, error)
}

// RedisSubscription is the driver-side subscription the RedisBroker adapts onto
// the pubsub.Subscription port. Concrete clients typically expose a receive
// channel (go-redis' PubSub.Channel()); the adapter forwards it.
type RedisSubscription interface {
	// Payloads is the stream of raw message bodies delivered to the subscription.
	Payloads() <-chan []byte
	// Close ends the subscription.
	Close() error
}

// RedisBroker is the production Broker: it fans messages out through Redis
// pub/sub so subscribers on every replica receive a payload published by any
// replica, giving consistent delivery across a horizontally-scaled gateway.
type RedisBroker struct {
	client RedisClient
}

// NewRedisBroker builds a Redis-backed broker over a narrow client.
func NewRedisBroker(client RedisClient) *RedisBroker {
	return &RedisBroker{client: client}
}

// Publish forwards to the underlying Redis client.
func (b *RedisBroker) Publish(ctx context.Context, channel string, payload []byte) error {
	return b.client.Publish(ctx, channel, payload)
}

// Subscribe opens a Redis subscription and adapts it onto the Subscription port,
// translating the driver's raw payload stream into pubsub.Message values.
func (b *RedisBroker) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	rs, err := b.client.Subscribe(ctx, channel)
	if err != nil {
		return nil, err
	}
	sub := &redisSubscription{
		inner:   rs,
		channel: channel,
		out:     make(chan Message),
		done:    make(chan struct{}),
	}
	go sub.pump()
	return sub, nil
}

// redisSubscription bridges a driver RedisSubscription onto pubsub.Subscription.
type redisSubscription struct {
	inner   RedisSubscription
	channel string
	out     chan Message
	done    chan struct{}
}

// pump forwards raw driver payloads onto the typed message channel until the
// driver stream drains or the subscription is closed.
func (s *redisSubscription) pump() {
	defer close(s.out)
	in := s.inner.Payloads()
	for {
		select {
		case <-s.done:
			return
		case payload, ok := <-in:
			if !ok {
				return
			}
			select {
			case s.out <- Message{Channel: s.channel, Payload: payload}:
			case <-s.done:
				return
			}
		}
	}
}

func (s *redisSubscription) C() <-chan Message { return s.out }

// Close stops the pump and closes the driver subscription.
func (s *redisSubscription) Close() error {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return s.inner.Close()
}

// Compile-time assertions that the Redis adapter satisfies the ports.
var (
	_ Broker       = (*RedisBroker)(nil)
	_ Subscription = (*redisSubscription)(nil)
)
