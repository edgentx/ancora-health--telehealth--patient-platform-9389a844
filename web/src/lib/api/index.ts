/**
 * Frontend API client layer — public entry point.
 *
 * View stories import from `@/lib/api` (hooks) and `@/lib/api` types; only the
 * transport internals live in the submodules. This barrel re-exports the client
 * factory, error types, realtime client, models and every bounded-context hook.
 */
export { apiClient, getApiClient, createApiClient } from './client';
export type { ApiClient, ApiClientOptions } from './client';

export { ApiError, isRetryableError } from './errors';
export type { ApiErrorKind, GraphQLResponseError } from './errors';

export { HttpClient, defaultConfig } from './http';
export type { HttpClientConfig, RequestOptions } from './http';

export { GraphQLClient } from './graphql';
export type { GraphQLRequest } from './graphql';

export { RealtimeClient, defaultRealtimeConfig } from './websocket';
export type {
  RealtimeChannel,
  RealtimeEnvelope,
  ConnectionStatus,
  SignalingChannel,
} from './websocket';

export { newTraceContext } from './trace';
export type { TraceContext } from './trace';

export * from './models';
export * from './hooks';
