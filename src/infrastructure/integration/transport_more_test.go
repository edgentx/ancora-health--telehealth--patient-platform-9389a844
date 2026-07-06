package integration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

// stubDoer is an in-process HTTPDoer whose behaviour a test controls.
type stubDoer struct {
	resp   *http.Response
	err    error
	called int
	lastFn func(*http.Request)
}

func (d *stubDoer) Do(req *http.Request) (*http.Response, error) {
	d.called++
	if d.lastFn != nil {
		d.lastFn(req)
	}
	return d.resp, d.err
}

// errReadCloser is a response body whose Read always fails, exercising the
// attempt's body-buffering error path.
type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReadCloser) Close() error             { return nil }

// spyRecorder records the accesses forwarded to it, optionally failing.
type spyRecorder struct {
	entries []OutboundAccess
	err     error
}

func (s *spyRecorder) RecordOutboundAccess(_ context.Context, a OutboundAccess) error {
	s.entries = append(s.entries, a)
	return s.err
}

func TestRecordIfSet(t *testing.T) {
	// A nil recorder is a tolerated no-op.
	if err := RecordIfSet(context.Background(), nil, OutboundAccess{}); err != nil {
		t.Fatalf("nil recorder returned %v, want nil", err)
	}

	// A non-nil recorder is forwarded to and its error surfaced.
	spy := &spyRecorder{}
	access := OutboundAccess{Action: "x.do", ResourceRef: "r-1"}
	if err := RecordIfSet(context.Background(), spy, access); err != nil {
		t.Fatalf("RecordIfSet: %v", err)
	}
	if len(spy.entries) != 1 || spy.entries[0].Action != "x.do" {
		t.Fatalf("recorder not forwarded: %+v", spy.entries)
	}

	sentinel := errors.New("sink down")
	failing := &spyRecorder{err: sentinel}
	if err := RecordIfSet(context.Background(), failing, access); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sink error", err)
	}
}

func TestConfigWithDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   Config
		want Config
	}{
		{
			name: "zero values fall back to defaults",
			in:   Config{},
			want: Config{Timeout: DefaultTimeout, MaxAttempts: DefaultMaxAttempts, Backoff: DefaultBackoff},
		},
		{
			name: "negative values fall back to defaults",
			in:   Config{Timeout: -1, MaxAttempts: -5, Backoff: -1},
			want: Config{Timeout: DefaultTimeout, MaxAttempts: DefaultMaxAttempts, Backoff: DefaultBackoff},
		},
		{
			name: "explicit values are preserved",
			in:   Config{BaseURL: "http://x", Timeout: 2 * time.Second, MaxAttempts: 7, Backoff: 3 * time.Millisecond},
			want: Config{BaseURL: "http://x", Timeout: 2 * time.Second, MaxAttempts: 7, Backoff: 3 * time.Millisecond},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.withDefaults()
			if got != tc.want {
				t.Fatalf("withDefaults() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestNewClient_NilDoerUsesDefaultHTTPClient(t *testing.T) {
	c := NewClient(nil, Config{})
	if c == nil || c.doer == nil {
		t.Fatal("NewClient(nil, ...) must supply a fallback doer")
	}
	if _, ok := c.doer.(*http.Client); !ok {
		t.Fatalf("fallback doer = %T, want *http.Client", c.doer)
	}
	// Defaults must have been filled in.
	if c.cfg.MaxAttempts != DefaultMaxAttempts {
		t.Fatalf("cfg not defaulted: %+v", c.cfg)
	}
}

func TestStatusError_Error(t *testing.T) {
	e := &StatusError{StatusCode: 503, Body: []byte("down")}
	if got := e.Error(); got != "integration: upstream returned status 503" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		raw     string
		want    string
	}{
		{"absolute http used verbatim", "http://base", "http://other/x", "http://other/x"},
		{"absolute https used verbatim", "http://base", "https://other/x", "https://other/x"},
		{"empty base returns raw", "", "/path", "/path"},
		{"rooted path joined to base", "http://base", "/path", "http://base/path"},
		{"trailing slash trimmed", "http://base/", "/path", "http://base/path"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{cfg: Config{BaseURL: tc.baseURL}}
			if got := c.resolveURL(tc.raw); got != tc.want {
				t.Fatalf("resolveURL(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestSleepCtx(t *testing.T) {
	// Elapses normally and returns nil.
	if err := sleepCtx(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("sleepCtx: %v", err)
	}

	// A cancelled context returns its error before the timer fires.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepCtx(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestSend_RequestBuildErrorIsTransport(t *testing.T) {
	doer := &stubDoer{}
	c := NewClient(doer, Config{MaxAttempts: 1, Backoff: time.Millisecond})
	c.sleep = func(context.Context, time.Duration) error { return nil }

	// An invalid method makes http.NewRequestWithContext fail before Do is called.
	_, err := c.Send(context.Background(), &Request{Method: "bad method", URL: "http://x"})
	if !errors.Is(err, ErrTransport) {
		t.Fatalf("err = %v, want ErrTransport", err)
	}
	if doer.called != 0 {
		t.Fatalf("doer called %d times, want 0 (request never built)", doer.called)
	}
}

func TestAttempt_PropagatesHeadersAndBody(t *testing.T) {
	var gotMethod, gotHeader string
	var gotBody []byte
	doer := &stubDoer{
		resp: &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: http.Header{}},
		lastFn: func(req *http.Request) {
			gotMethod = req.Method
			gotHeader = req.Header.Get("X-Test")
			gotBody, _ = io.ReadAll(req.Body)
		},
	}
	c := NewClient(doer, Config{MaxAttempts: 1})
	c.sleep = noSleep

	resp, err := c.Send(context.Background(), &Request{
		Method: http.MethodPost,
		URL:    "http://x/y",
		Header: http.Header{"X-Test": []string{"v"}},
		Body:   []byte("payload"),
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if gotMethod != http.MethodPost || gotHeader != "v" || string(gotBody) != "payload" {
		t.Fatalf("request not built as expected: method=%q header=%q body=%q", gotMethod, gotHeader, gotBody)
	}
}

func TestAttempt_BodyReadErrorIsTransport(t *testing.T) {
	doer := &stubDoer{resp: &http.Response{StatusCode: http.StatusOK, Body: errReadCloser{}, Header: http.Header{}}}
	c := NewClient(doer, Config{MaxAttempts: 1})
	c.sleep = noSleep

	_, err := c.Send(context.Background(), &Request{Method: http.MethodGet, URL: "http://x"})
	if !errors.Is(err, ErrTransport) {
		t.Fatalf("err = %v, want ErrTransport on body read failure", err)
	}
}

func TestSend_SleepErrorAborts(t *testing.T) {
	doer := &stubDoer{resp: &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody, Header: http.Header{}}}
	c := NewClient(doer, Config{MaxAttempts: 3, Backoff: time.Millisecond})
	sentinel := errors.New("sleep interrupted")
	c.sleep = func(context.Context, time.Duration) error { return sentinel }

	_, err := c.Send(context.Background(), &Request{Method: http.MethodGet, URL: "http://x"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sleep sentinel", err)
	}
	if doer.called != 1 {
		t.Fatalf("doer called %d times, want 1 (aborted before retry)", doer.called)
	}
}
