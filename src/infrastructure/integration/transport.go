// Package integration provides the shared HTTP transport the outbound
// integration adapters (pharmacy gateway, payment gateway, payer eligibility,
// object storage) are built on.
//
// Every adapter talks to a third-party service over HTTP, and every one needs
// the same operational envelope: a bounded timeout per attempt, a small number
// of retries with exponential backoff on transient failures, and a single,
// structured way to surface upstream errors to the domain. Rather than each
// adapter reimplementing that envelope against *http.Client, they depend on the
// narrow HTTPDoer seam and the Client here — the same approach the locking
// package takes with RedisConn, keeping concrete SDK dependencies out of the
// module.
package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPDoer is the narrow slice of an HTTP client the transport depends on. The
// standard library's *http.Client satisfies it, and tests supply an in-process
// stub (or an httptest.Server's client) so adapter behaviour — retries,
// timeouts, signature verification, error mapping — can be exercised without a
// real network. Isolating this one method keeps the transport free of any
// dependency on a specific HTTP client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

const (
	// DefaultTimeout bounds a single request attempt. It is applied as a
	// per-attempt context deadline so a hung upstream cannot wedge a caller.
	DefaultTimeout = 10 * time.Second
	// DefaultMaxAttempts is the total number of attempts (the first try plus
	// retries) made for a retryable failure.
	DefaultMaxAttempts = 3
	// DefaultBackoff is the base delay for exponential backoff between attempts.
	DefaultBackoff = 100 * time.Millisecond
	// maxResponseBytes caps how much of an upstream response body is read into
	// memory, bounding the blast radius of a hostile or malfunctioning upstream.
	maxResponseBytes = 8 << 20 // 8 MiB
)

// Config captures the operational settings shared by every adapter's transport.
// The zero value is not usable; build one with sensible fields or rely on the
// defaults NewClient fills in.
type Config struct {
	// BaseURL is the upstream service root; adapter request paths are resolved
	// against it.
	BaseURL string
	// Timeout bounds a single attempt. A non-positive value falls back to
	// DefaultTimeout.
	Timeout time.Duration
	// MaxAttempts is the total number of attempts for a retryable failure. A
	// value below 1 falls back to DefaultMaxAttempts.
	MaxAttempts int
	// Backoff is the base delay for exponential backoff. A non-positive value
	// falls back to DefaultBackoff.
	Backoff time.Duration
}

func (c Config) withDefaults() Config {
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.MaxAttempts < 1 {
		c.MaxAttempts = DefaultMaxAttempts
	}
	if c.Backoff <= 0 {
		c.Backoff = DefaultBackoff
	}
	return c
}

// Client is the retrying, timeout-bounded HTTP transport the adapters send
// through. It wraps an HTTPDoer with the shared operational envelope and returns
// structured errors the adapters map to their own domain-facing failures.
type Client struct {
	doer HTTPDoer
	cfg  Config
	// sleep abstracts the backoff wait so tests can run retry paths without real
	// delays. It defaults to a context-aware time.After wait.
	sleep func(ctx context.Context, d time.Duration) error
}

// NewClient builds a transport over doer with cfg. A nil doer falls back to a
// timeout-less http.Client (the per-attempt context deadline is the real
// bound); missing config fields fall back to the package defaults.
func NewClient(doer HTTPDoer, cfg Config) *Client {
	if doer == nil {
		doer = &http.Client{}
	}
	return &Client{
		doer:  doer,
		cfg:   cfg.withDefaults(),
		sleep: sleepCtx,
	}
}

// Request is a transport-level request. Body is carried as bytes rather than an
// io.Reader so the transport can replay it verbatim on every retry attempt.
type Request struct {
	Method string
	// URL is the absolute request URL. Adapters build it from Config.BaseURL and
	// their own path/query so the transport stays agnostic to service layout.
	URL    string
	Header http.Header
	Body   []byte
}

// Response is the completed round-trip the transport hands back. Body is fully
// buffered (bounded by maxResponseBytes) so adapters can parse or verify it
// without worrying about stream lifetimes.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// StatusError is returned by Send when an upstream responds with a status of 400
// or greater. It preserves the code and a bounded snippet of the body so
// adapters can map specific upstream failures to domain errors while callers can
// still match it with errors.As.
type StatusError struct {
	StatusCode int
	Body       []byte
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("integration: upstream returned status %d", e.StatusCode)
}

// ErrTransport wraps a network- or client-level failure (connection refused,
// DNS, per-attempt timeout) that persisted across every retry attempt. Adapters
// match it with errors.Is to distinguish "we never got an answer" from "the
// upstream answered with an error status".
var ErrTransport = errors.New("integration: transport failure")

// Send performs req through the retrying transport and returns the completed
// response. Retryable failures — network errors, per-attempt timeouts, HTTP 429,
// and any 5xx — are retried up to Config.MaxAttempts with exponential backoff;
// context cancellation aborts immediately. A completed round trip whose status
// is >= 400 is returned alongside a *StatusError so callers get both the parsed
// body and a typed error; a 2xx returns a nil error. Exhausting retries on a
// transport failure returns an error wrapping ErrTransport.
func (c *Client) Send(ctx context.Context, req *Request) (*Response, error) {
	var lastErr error
	for attempt := 1; attempt <= c.cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		resp, err := c.attempt(ctx, req)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrTransport, err)
		} else if isRetryableStatus(resp.StatusCode) {
			lastErr = &StatusError{StatusCode: resp.StatusCode, Body: resp.Body}
		} else if resp.StatusCode >= http.StatusBadRequest {
			// A non-retryable error status (4xx other than 429) is final: return
			// the response so the adapter can read the body, plus the typed error.
			return resp, &StatusError{StatusCode: resp.StatusCode, Body: resp.Body}
		} else {
			return resp, nil
		}

		if attempt == c.cfg.MaxAttempts {
			break
		}
		if err := c.sleep(ctx, backoff(c.cfg.Backoff, attempt)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

// attempt performs a single HTTP round trip under a per-attempt timeout and
// buffers the response body.
func (c *Client) attempt(ctx context.Context, req *Request) (*Response, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(attemptCtx, req.Method, c.resolveURL(req.URL), body)
	if err != nil {
		return nil, err
	}
	for k, vs := range req.Header {
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
	}

	httpResp, err := c.doer.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	buf, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBytes))
	if err != nil {
		return nil, err
	}
	return &Response{
		StatusCode: httpResp.StatusCode,
		Header:     httpResp.Header,
		Body:       buf,
	}, nil
}

// resolveURL joins a request URL against the configured BaseURL. An absolute URL
// (already carrying a scheme, as the object-storage adapter builds) is used
// verbatim; a rooted path is resolved against BaseURL so adapters can address
// their service by path alone.
func (c *Client) resolveURL(raw string) string {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if c.cfg.BaseURL == "" {
		return raw
	}
	return strings.TrimSuffix(c.cfg.BaseURL, "/") + raw
}

// isRetryableStatus reports whether an HTTP status warrants a retry: 429 (rate
// limited) and any 5xx (server-side, presumed transient).
func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
}

// backoff computes the exponential delay before the next attempt (attempt is
// 1-based): base, 2·base, 4·base, ...
func backoff(base time.Duration, attempt int) time.Duration {
	return base * (1 << (attempt - 1))
}

// sleepCtx waits for d or until ctx is cancelled, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
