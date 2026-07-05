/**
 * Centralised environment configuration.
 *
 * API and WebSocket base URLs are browser-visible (NEXT_PUBLIC_*) because the
 * client issues fetches and opens sockets directly. The trusted-header names are
 * server-only: they are read while resolving identity in server components and
 * must never leak to the bundle.
 */

/** Browser-visible endpoint config. Safe to reference from client components. */
export const publicEnv = {
  apiBaseUrl: process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8000',
  wsBaseUrl: process.env.NEXT_PUBLIC_WS_BASE_URL ?? 'ws://localhost:8000',
} as const;

/**
 * Server-only config for the trusted identity headers surfaced by the edge.
 * Importing this into a client component is a build-time error waiting to happen,
 * so keep its use confined to server code (see {@link ./identity}).
 */
export const serverEnv = {
  roleHeader: process.env.ANCORA_ROLE_HEADER ?? 'x-ancora-role',
  userHeader: process.env.ANCORA_USER_HEADER ?? 'x-ancora-user',
} as const;
