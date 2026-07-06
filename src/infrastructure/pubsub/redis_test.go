package pubsub

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeRedisSub is an in-process RedisSubscription: payloads pushed onto its
// channel are surfaced by Payloads(). It records whether Close was called.
type fakeRedisSub struct {
	payloads chan []byte
	closed   bool
}

func (f *fakeRedisSub) Payloads() <-chan []byte { return f.payloads }
func (f *fakeRedisSub) Close() error {
	f.closed = true
	return nil
}

// fakeRedisClient is an in-process RedisClient standing in for a driver, so the
// RedisBroker adapter can be exercised without a live Redis.
type fakeRedisClient struct {
	subErr  error
	pubErr  error
	sub     *fakeRedisSub
	lastCh  string
	lastPay []byte
}

func (f *fakeRedisClient) Publish(_ context.Context, channel string, payload []byte) error {
	f.lastCh = channel
	f.lastPay = payload
	return f.pubErr
}

func (f *fakeRedisClient) Subscribe(_ context.Context, _ string) (RedisSubscription, error) {
	if f.subErr != nil {
		return nil, f.subErr
	}
	return f.sub, nil
}

func TestMemoryBroker_DefaultBuffer(t *testing.T) {
	// A non-positive buffer falls back to the default and still delivers.
	for _, buf := range []int{0, -5} {
		broker := NewMemoryBroker(buf)
		sub, err := broker.Subscribe(context.Background(), "c")
		if err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		if err := broker.Publish(context.Background(), "c", []byte("x")); err != nil {
			t.Fatalf("publish: %v", err)
		}
		select {
		case m := <-sub.C():
			if string(m.Payload) != "x" {
				t.Fatalf("payload = %q", m.Payload)
			}
		case <-time.After(time.Second):
			t.Fatal("no delivery with default buffer")
		}
		sub.Close()
	}
}

func TestRedisBroker_Publish(t *testing.T) {
	client := &fakeRedisClient{}
	b := NewRedisBroker(client)
	if err := b.Publish(context.Background(), "ch", []byte("hi")); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if client.lastCh != "ch" || string(client.lastPay) != "hi" {
		t.Fatalf("forwarded %q/%q", client.lastCh, client.lastPay)
	}

	client.pubErr = errors.New("down")
	if err := b.Publish(context.Background(), "ch", nil); err == nil {
		t.Fatal("expected publish error to propagate")
	}
}

func TestRedisBroker_SubscribeError(t *testing.T) {
	b := NewRedisBroker(&fakeRedisClient{subErr: errors.New("no")})
	if _, err := b.Subscribe(context.Background(), "ch"); err == nil {
		t.Fatal("expected subscribe error")
	}
}

func TestRedisBroker_SubscribeDeliversAndCloses(t *testing.T) {
	rs := &fakeRedisSub{payloads: make(chan []byte, 1)}
	b := NewRedisBroker(&fakeRedisClient{sub: rs})

	sub, err := b.Subscribe(context.Background(), "ch")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	rs.payloads <- []byte("payload")
	select {
	case m := <-sub.C():
		if m.Channel != "ch" || string(m.Payload) != "payload" {
			t.Fatalf("delivered %q/%q", m.Channel, m.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("no message forwarded by pump")
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if !rs.closed {
		t.Fatal("expected the driver subscription to be closed")
	}
	// Close is idempotent (second call takes the already-done branch).
	if err := sub.Close(); err != nil {
		t.Fatalf("double close: %v", err)
	}
}

func TestRedisBroker_PumpStopsWhenDriverStreamCloses(t *testing.T) {
	rs := &fakeRedisSub{payloads: make(chan []byte)}
	b := NewRedisBroker(&fakeRedisClient{sub: rs})

	sub, err := b.Subscribe(context.Background(), "ch")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	// Closing the driver's payload stream makes pump drain and close its output.
	close(rs.payloads)
	select {
	case _, ok := <-sub.C():
		if ok {
			t.Fatal("expected the message channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("pump did not close its output when the driver stream closed")
	}
}

func TestRedisBroker_CloseUnblocksPendingSend(t *testing.T) {
	rs := &fakeRedisSub{payloads: make(chan []byte, 1)}
	b := NewRedisBroker(&fakeRedisClient{sub: rs})

	sub, err := b.Subscribe(context.Background(), "ch")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Push a payload but never read sub.C(); the pump blocks trying to forward it
	// on the unbuffered output. Close must unblock the pump via its done channel.
	rs.payloads <- []byte("stuck")
	time.Sleep(20 * time.Millisecond)
	if err := sub.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
