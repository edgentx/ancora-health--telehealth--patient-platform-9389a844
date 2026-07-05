// Package rest is the REST handler layer that exposes the domain use cases of
// every bounded context over HTTP. It owns route registration and API
// versioning, JSON request/response DTO mapping, input validation, structured
// error responses, pagination/filtering, and the observability middleware
// (OpenTelemetry tracing + a Prometheus /metrics endpoint).
//
// The layer is intentionally thin: it maps transport concerns to domain
// commands/queries and back, and holds no authentication or authorization
// logic. Caller identity, role and tenant arrive as trusted headers injected by
// the Kong + OPA edge; handlers read them and pass them through to the domain,
// but never validate JWTs, evaluate RBAC, or resolve tenants themselves.
package rest

import (
	"context"
	"net/http"
	"strings"
)

// Trusted-identity headers. The Kong + OPA edge authenticates the caller and
// resolves its tenant before the request ever reaches this service, then stamps
// the result onto these headers. Because the edge is the only ingress, the
// handler layer treats them as trusted facts — it reads them, it does not
// re-derive or verify them.
const (
	// HeaderSubject carries the authenticated caller's stable subject identifier.
	HeaderSubject = "X-Identity-Subject"
	// HeaderRoles carries the caller's roles as a comma-separated list.
	HeaderRoles = "X-Identity-Roles"
	// HeaderTenant carries the tenant the request is scoped to.
	HeaderTenant = "X-Tenant-Id"
)

// Caller is the trusted identity context of a request as resolved by the edge.
// It is a read-only carrier: the handler layer threads it into domain
// commands/queries (for example as an actor reference on an audit entry) but
// never makes an access-control decision from it.
type Caller struct {
	// Subject is the authenticated caller's stable identifier.
	Subject string
	// Roles are the roles the edge asserted for the caller.
	Roles []string
	// Tenant is the tenant the request is scoped to.
	Tenant string
}

// HasRole reports whether the caller was asserted to hold role. It exists so a
// handler can attach role-derived context to a command (never to gate access —
// authorization is decided at the edge).
func (c Caller) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// callerContextKey is the private context key the resolved Caller is stored
// under, so only this package can set or retrieve it.
type callerContextKey struct{}

// IdentityMiddleware parses the trusted edge headers into a Caller and stashes
// it on the request context for downstream handlers. It performs no validation
// or rejection: an absent header simply yields an empty field, because deciding
// whether a caller may proceed is the edge's job, not this layer's.
func IdentityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller := Caller{
			Subject: strings.TrimSpace(r.Header.Get(HeaderSubject)),
			Roles:   parseRoles(r.Header.Get(HeaderRoles)),
			Tenant:  strings.TrimSpace(r.Header.Get(HeaderTenant)),
		}
		ctx := context.WithValue(r.Context(), callerContextKey{}, caller)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CallerFrom returns the Caller resolved by IdentityMiddleware. When the
// middleware did not run (or set nothing) it returns the zero Caller, so
// handlers can always read it without a nil check.
func CallerFrom(ctx context.Context) Caller {
	if c, ok := ctx.Value(callerContextKey{}).(Caller); ok {
		return c
	}
	return Caller{}
}

// parseRoles splits a comma-separated roles header into a trimmed, non-empty
// slice. An empty or whitespace-only header yields no roles.
func parseRoles(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	roles := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			roles = append(roles, trimmed)
		}
	}
	return roles
}
