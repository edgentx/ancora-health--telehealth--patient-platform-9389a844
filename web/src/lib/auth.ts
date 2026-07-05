import { publicEnv } from './env';

/**
 * Edge-managed authentication seam.
 *
 * The app does not authenticate users itself — it delegates to the Kong+OPA
 * edge, which owns the login flow (OIDC/session/MFA) and, on success, re-issues
 * the request with the trusted identity headers this app reads server-side (see
 * {@link ./identity}). These helpers are the *only* place the client touches the
 * auth flow, and all they do is navigate the browser to an edge endpoint. There
 * is deliberately no token parsing, password handling, or RBAC here.
 *
 * `beginEdgeLogin` / `beginEdgeLogout` are isolated one-liners so components can
 * be unit-tested by mocking this module instead of stubbing `window.location`.
 */

/** Append a `return_to` hint the edge honours after the flow completes. */
function withReturnTo(base: string, returnTo?: string): string {
  if (!returnTo) return base;
  const sep = base.includes('?') ? '&' : '?';
  return `${base}${sep}return_to=${encodeURIComponent(returnTo)}`;
}

/** URL that starts the edge login flow, optionally returning to `returnTo`. */
export function edgeLoginUrl(returnTo?: string): string {
  return withReturnTo(publicEnv.edgeLoginUrl, returnTo);
}

/** URL that ends the edge session, optionally returning to `returnTo`. */
export function edgeLogoutUrl(returnTo?: string): string {
  return withReturnTo(publicEnv.edgeLogoutUrl, returnTo);
}

/** Navigate the browser into the edge login flow. */
export function beginEdgeLogin(returnTo?: string): void {
  window.location.assign(edgeLoginUrl(returnTo));
}

/** Navigate the browser to the edge logout endpoint. */
export function beginEdgeLogout(returnTo?: string): void {
  window.location.assign(edgeLogoutUrl(returnTo));
}
