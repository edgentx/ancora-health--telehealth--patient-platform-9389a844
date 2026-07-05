package integration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// noSleep replaces the backoff wait so retry paths run without real delays.
func noSleep(_ context.Context, _ time.Duration) error { return nil }

func TestClient_RetriesOnServerErrorThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), Config{BaseURL: srv.URL, MaxAttempts: 3, Backoff: time.Millisecond})
	c.sleep = noSleep

	resp, err := c.Send(context.Background(), &Request{Method: http.MethodGet, URL: "/thing"})
	if err != nil {
		t.Fatalf("Send after retries: %v", err)
	}
	if resp.StatusCode != http.StatusOK || string(resp.Body) != "ok" {
		t.Fatalf("resp = %d %q", resp.StatusCode, resp.Body)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
}

func TestClient_ExhaustsRetriesOnPersistentServerError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), Config{BaseURL: srv.URL, MaxAttempts: 3, Backoff: time.Millisecond})
	c.sleep = noSleep

	_, err := c.Send(context.Background(), &Request{Method: http.MethodGet, URL: "/thing"})
	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("err = %v, want StatusError 500", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls = %d, want 3 (all attempts used)", got)
	}
}

func TestClient_DoesNotRetryClientError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), Config{BaseURL: srv.URL, MaxAttempts: 3, Backoff: time.Millisecond})
	c.sleep = noSleep

	resp, err := c.Send(context.Background(), &Request{Method: http.MethodGet, URL: "/thing"})
	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %v, want StatusError 400", err)
	}
	if resp == nil || string(resp.Body) != "nope" {
		t.Fatalf("resp body not surfaced on 4xx: %+v", resp)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls = %d, want 1 (no retry on 4xx)", got)
	}
}

func TestClient_TimeoutIsRetriedThenReportedAsTransport(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), Config{BaseURL: srv.URL, MaxAttempts: 2, Backoff: time.Millisecond, Timeout: 5 * time.Millisecond})
	c.sleep = noSleep

	_, err := c.Send(context.Background(), &Request{Method: http.MethodGet, URL: "/slow"})
	if !errors.Is(err, ErrTransport) {
		t.Fatalf("err = %v, want ErrTransport (per-attempt timeout)", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2 (timeout retried)", got)
	}
}

func TestClient_ContextCancellationAborts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), Config{BaseURL: srv.URL, MaxAttempts: 5, Backoff: time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	_, err := c.Send(ctx, &Request{Method: http.MethodGet, URL: "/thing"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
