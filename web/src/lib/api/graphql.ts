/**
 * Typed GraphQL transport.
 *
 * The backend serves gqlgen at a single POST endpoint behind the edge. This
 * client reuses {@link HttpClient} so GraphQL inherits the exact same base URL,
 * credentials, trace propagation, timeout and retry/backoff as REST — the only
 * GraphQL-specific concern is unwrapping `{ data, errors }` and turning a
 * populated `errors` array into an {@link ApiError}, since GraphQL reports
 * failures with a 200 status.
 */

import { publicEnv } from '@/lib/env';

import { ApiError, normalizeGraphQLError, type GraphQLResponseError } from './errors';
import type { HttpClient } from './http';

/** A GraphQL operation: a document string plus typed variables. */
export interface GraphQLRequest<V extends Record<string, unknown>> {
  query: string;
  variables?: V;
  /** Operation name, forwarded for server-side logging/APQ. */
  operationName?: string;
}

interface GraphQLEnvelope<T> {
  data?: T;
  errors?: GraphQLResponseError[];
}

/** GraphQL client bound to a REST {@link HttpClient} for shared transport. */
export class GraphQLClient {
  constructor(
    private readonly http: HttpClient,
    private readonly path: string = publicEnv.graphqlPath,
  ) {}

  /**
   * Execute an operation, returning typed `data`. Rejects with an
   * {@link ApiError} of kind `graphql` when the response carries `errors`, or a
   * transport-kind error if the request itself fails.
   */
  async execute<T, V extends Record<string, unknown> = Record<string, unknown>>(
    operation: GraphQLRequest<V>,
  ): Promise<T> {
    // GraphQL mutations are non-idempotent; never auto-retry a POST here.
    const envelope = await this.http.post<GraphQLEnvelope<T>>(
      this.path,
      {
        query: operation.query,
        variables: operation.variables ?? {},
        operationName: operation.operationName,
      },
      { retry: false },
    );

    if (envelope.errors && envelope.errors.length > 0) {
      throw normalizeGraphQLError(envelope.errors);
    }
    if (envelope.data === undefined) {
      throw new ApiError({
        kind: 'graphql',
        message: 'GraphQL response contained no data',
      });
    }
    return envelope.data;
  }
}
