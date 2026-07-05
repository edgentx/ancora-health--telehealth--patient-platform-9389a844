/**
 * Centralised environment configuration.
 *
 * API and WebSocket base URLs are browser-visible (NEXT_PUBLIC_*) because the
 * client issues fetches and opens sockets directly. The trusted-header names are
 * server-only: they are read while resolving identity in server components and
 * must never leak to the bundle.
 */

/**
 * Project slug used to derive the realtime host (`ws.{project}.vforce360.ai`)
 * when no explicit `NEXT_PUBLIC_WS_BASE_URL` override is supplied. Kept public
 * because the browser opens the socket directly.
 */
const projectSlug = process.env.NEXT_PUBLIC_PROJECT_SLUG ?? 'ancora';

/** Browser-visible endpoint config. Safe to reference from client components. */
export const publicEnv = {
  /** Project slug, e.g. `ancora`. Drives the default realtime host. */
  projectSlug,
  /** REST + GraphQL base URL, routed through the Kong edge. */
  apiBaseUrl: process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8000',
  /**
   * WebSocket base URL for realtime messaging/notifications and WebRTC
   * signaling. Defaults to the per-project edge host `ws.{project}.vforce360.ai`
   * in deployed environments; a local override points it at the dev server.
   */
  wsBaseUrl:
    process.env.NEXT_PUBLIC_WS_BASE_URL ?? `wss://ws.${projectSlug}.vforce360.ai`,
  /**
   * GraphQL path appended to {@link apiBaseUrl}. The backend serves gqlgen at a
   * single endpoint behind the edge.
   */
  graphqlPath: process.env.NEXT_PUBLIC_GRAPHQL_PATH ?? '/graphql',
  /**
   * Service name advertised on outbound OpenTelemetry baggage so the edge and
   * backend can attribute client-originated spans.
   */
  otelServiceName: process.env.NEXT_PUBLIC_OTEL_SERVICE_NAME ?? 'ancora-web',
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
