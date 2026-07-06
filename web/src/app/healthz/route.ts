/**
 * Liveness/readiness probe endpoint for the web tier (S-83).
 *
 * Kubernetes probes (deploy/helm/ancora) hit `/healthz`. It is deliberately
 * unauthenticated and dependency-free: it proves the Next.js server process is
 * up and serving, independent of the edge identity headers every app page
 * requires — a probe must never depend on a logged-in role.
 */

// Always execute on request; never statically cache the health response.
export const dynamic = 'force-dynamic';

export function GET(): Response {
  return Response.json({ status: 'ok', service: 'ancora-web' });
}
