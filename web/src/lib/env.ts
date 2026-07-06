/**
 * Centralised environment configuration.
 *
 * Endpoint config (API/WS base URLs, GraphQL path, edge login routes) is a
 * *runtime* concern: one built image must retarget to any environment purely
 * from the container's environment, with no rebuild (S-83). Next inlines
 * `NEXT_PUBLIC_*` at build time, so those values are frozen into the bundle and
 * cannot carry runtime config. Instead:
 *
 *   - On the server the config is resolved from `process.env` at request time,
 *     reading unprefixed runtime keys (`API_BASE_URL`, `WS_BASE_URL`, …).
 *   - The root layout serialises that snapshot into the document via
 *     {@link serializeRuntimeEnvScript}, seeding {@link RUNTIME_ENV_GLOBAL} on
 *     `window` before hydration.
 *   - On the browser {@link publicEnv} reads that injected snapshot, falling back
 *     to the build-time `NEXT_PUBLIC_*` defaults only when it is absent (tests).
 *
 * The trusted-header names ({@link serverEnv}) are server-only and read straight
 * from `process.env`; they must never leak into the client bundle.
 */

/** Browser-visible endpoint configuration resolved for the current environment. */
export interface PublicConfig {
  /** Project slug, e.g. `ancora`. Drives the default realtime host. */
  projectSlug: string;
  /** REST + GraphQL base URL, routed through the Kong edge. */
  apiBaseUrl: string;
  /** WebSocket base URL for realtime messaging/notifications and WebRTC. */
  wsBaseUrl: string;
  /** GraphQL path appended to {@link apiBaseUrl}. */
  graphqlPath: string;
  /** Service name advertised on outbound OpenTelemetry baggage. */
  otelServiceName: string;
  /** Edge endpoint that starts the managed login flow. */
  edgeLoginUrl: string;
  /** Edge endpoint that ends the session. */
  edgeLogoutUrl: string;
}

/** Name of the browser global the server seeds with the runtime {@link PublicConfig}. */
export const RUNTIME_ENV_GLOBAL = '__ANCORA_ENV__' as const;

declare global {
  interface Window {
    [RUNTIME_ENV_GLOBAL]?: PublicConfig;
  }
}

type EnvSource = Record<string, string | undefined>;

/**
 * Pure resolver: derive the browser-visible config from an env-like source,
 * applying one set of defaults everywhere. Each field prefers its unprefixed
 * *runtime* key (set on the deployed container), then the build-time
 * `NEXT_PUBLIC_*` value, then a local-development default.
 */
export function resolvePublicConfig(source: EnvSource): PublicConfig {
  const projectSlug = source.PROJECT_SLUG ?? source.NEXT_PUBLIC_PROJECT_SLUG ?? 'ancora';
  return {
    projectSlug,
    apiBaseUrl:
      source.API_BASE_URL ?? source.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8000',
    wsBaseUrl:
      source.WS_BASE_URL ??
      source.NEXT_PUBLIC_WS_BASE_URL ??
      `wss://ws.${projectSlug}.vforce360.ai`,
    graphqlPath: source.GRAPHQL_PATH ?? source.NEXT_PUBLIC_GRAPHQL_PATH ?? '/graphql',
    otelServiceName:
      source.OTEL_SERVICE_NAME ?? source.NEXT_PUBLIC_OTEL_SERVICE_NAME ?? 'ancora-web',
    edgeLoginUrl: source.EDGE_LOGIN_URL ?? source.NEXT_PUBLIC_EDGE_LOGIN_URL ?? '/auth/login',
    edgeLogoutUrl:
      source.EDGE_LOGOUT_URL ?? source.NEXT_PUBLIC_EDGE_LOGOUT_URL ?? '/auth/logout',
  };
}

/**
 * Build-time inlined fallback for the browser. Each `process.env.NEXT_PUBLIC_*`
 * reference is replaced with a literal by Next at build; this is used only when
 * the server-injected runtime snapshot is unavailable (e.g. unit tests).
 */
function inlinedPublicConfig(): PublicConfig {
  return resolvePublicConfig({
    NEXT_PUBLIC_PROJECT_SLUG: process.env.NEXT_PUBLIC_PROJECT_SLUG,
    NEXT_PUBLIC_API_BASE_URL: process.env.NEXT_PUBLIC_API_BASE_URL,
    NEXT_PUBLIC_WS_BASE_URL: process.env.NEXT_PUBLIC_WS_BASE_URL,
    NEXT_PUBLIC_GRAPHQL_PATH: process.env.NEXT_PUBLIC_GRAPHQL_PATH,
    NEXT_PUBLIC_OTEL_SERVICE_NAME: process.env.NEXT_PUBLIC_OTEL_SERVICE_NAME,
    NEXT_PUBLIC_EDGE_LOGIN_URL: process.env.NEXT_PUBLIC_EDGE_LOGIN_URL,
    NEXT_PUBLIC_EDGE_LOGOUT_URL: process.env.NEXT_PUBLIC_EDGE_LOGOUT_URL,
  });
}

/**
 * Resolve the effective config for the current execution context.
 *
 * - Server (SSR, server components, route handlers): read `process.env` *now*,
 *   so a value set on the container at boot is honoured without a rebuild.
 * - Browser: prefer the snapshot the server injected into the document; fall
 *   back to the build-time inlined values only when it is absent.
 */
function resolvePublicEnv(): PublicConfig {
  if (typeof window === 'undefined') {
    return resolvePublicConfig(process.env as EnvSource);
  }
  return window[RUNTIME_ENV_GLOBAL] ?? inlinedPublicConfig();
}

/**
 * Browser-visible endpoint config. Property access is resolved lazily so the
 * value reflects the runtime environment (server) or the injected snapshot
 * (browser) rather than being frozen at module load.
 */
export const publicEnv: PublicConfig = {
  get projectSlug() {
    return resolvePublicEnv().projectSlug;
  },
  get apiBaseUrl() {
    return resolvePublicEnv().apiBaseUrl;
  },
  get wsBaseUrl() {
    return resolvePublicEnv().wsBaseUrl;
  },
  get graphqlPath() {
    return resolvePublicEnv().graphqlPath;
  },
  get otelServiceName() {
    return resolvePublicEnv().otelServiceName;
  },
  get edgeLoginUrl() {
    return resolvePublicEnv().edgeLoginUrl;
  },
  get edgeLogoutUrl() {
    return resolvePublicEnv().edgeLogoutUrl;
  },
};

/**
 * Server-only: the runtime config snapshot to embed in the document so the
 * browser sees exactly what the server resolved for this request. Reads
 * `process.env` at call time — the root layout is dynamic, so this runs
 * per-request rather than at build.
 */
export function publicConfigSnapshot(): PublicConfig {
  return resolvePublicConfig(process.env as EnvSource);
}

/**
 * Serialise a {@link PublicConfig} into an inline `<script>` body that seeds
 * {@link RUNTIME_ENV_GLOBAL} before hydration. `<` is escaped so a value can
 * never break out of the script element.
 */
export function serializeRuntimeEnvScript(config: PublicConfig): string {
  const json = JSON.stringify(config).replace(/</g, '\\u003c');
  return `window.${RUNTIME_ENV_GLOBAL}=Object.freeze(${json});`;
}

/**
 * Server-only config for the trusted identity headers surfaced by the edge.
 * Importing this into a client component is a build-time error waiting to happen,
 * so keep its use confined to server code (see {@link ./identity}).
 */
export const serverEnv = {
  roleHeader: process.env.ANCORA_ROLE_HEADER ?? 'x-ancora-role',
  userHeader: process.env.ANCORA_USER_HEADER ?? 'x-ancora-user',
} as const;
