/**
 * The assembled API client.
 *
 * One place wires the REST transport, the GraphQL client (sharing that
 * transport), and the realtime WebSocket client. Application code imports the
 * lazily-created singleton {@link apiClient}; tests build isolated instances via
 * {@link createApiClient} so each spec gets a fresh, independently-configured
 * client against a mocked network.
 */

import { GraphQLClient } from './graphql';
import { defaultConfig, HttpClient, type HttpClientConfig } from './http';
import { defaultRealtimeConfig, RealtimeClient, type RealtimeConfig } from './websocket';

/** The public surface of the client: REST, GraphQL and realtime together. */
export interface ApiClient {
  rest: HttpClient;
  graphql: GraphQLClient;
  realtime: RealtimeClient;
}

/** Overrides for constructing a client, primarily for tests. */
export interface ApiClientOptions {
  http?: Partial<HttpClientConfig>;
  realtime?: Partial<RealtimeConfig>;
}

/** Build a fresh, fully-wired client. */
export function createApiClient(options: ApiClientOptions = {}): ApiClient {
  const rest = new HttpClient({ ...defaultConfig(), ...options.http });
  return {
    rest,
    graphql: new GraphQLClient(rest),
    realtime: new RealtimeClient({ ...defaultRealtimeConfig(), ...options.realtime }),
  };
}

let singleton: ApiClient | null = null;

/**
 * The process-wide client. Created lazily on first use so importing this module
 * during SSR/build never opens a socket or reads env before it is set.
 */
export function getApiClient(): ApiClient {
  if (!singleton) singleton = createApiClient();
  return singleton;
}

/** Convenience accessor mirroring the common `apiClient.rest.get(...)` usage. */
export const apiClient: ApiClient = new Proxy({} as ApiClient, {
  get(_target, prop: keyof ApiClient) {
    return getApiClient()[prop];
  },
});
