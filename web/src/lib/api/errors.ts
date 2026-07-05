/**
 * Centralized error handling for the API client.
 *
 * Every failure — a non-2xx REST response, a GraphQL `errors` payload, a
 * transport failure (offline, DNS, TLS), or a timeout — is funnelled into a
 * single {@link ApiError} shape so callers (and the TanStack Query retry
 * predicate) reason about one type. The crucial distinction the UI needs is
 * *retryable* (transient: try again / show a soft retry affordance) vs
 * *terminal* (a bad request, forbidden, not-found: surface and stop).
 */

/** Where a failure originated, for logging and UX branching. */
export type ApiErrorKind =
  | 'network' // transport failed before a response (offline, DNS, TLS, CORS)
  | 'timeout' // the request exceeded its deadline
  | 'http' // the server answered with a non-2xx status
  | 'graphql' // a 200 GraphQL response carrying an `errors` array
  | 'parse' // the body could not be decoded as expected
  | 'unknown';

/** HTTP statuses that are safe to retry (transient / server-side). */
const RETRYABLE_STATUS = new Set([408, 425, 429, 500, 502, 503, 504]);

/**
 * A backend error envelope. The Go services return small JSON objects; we probe
 * a few conventional field names rather than couple to one exact shape, since
 * different layers (edge, gqlgen, aggregate guards) phrase errors differently.
 */
interface BackendErrorBody {
  error?: string | { code?: string; message?: string; retryable?: boolean };
  code?: string;
  message?: string;
  reason?: string;
  detail?: string;
  retryable?: boolean;
}

/** Options for constructing an {@link ApiError}. */
interface ApiErrorInit {
  kind: ApiErrorKind;
  message: string;
  /** HTTP status, when a response was received. */
  status?: number;
  /** Machine-readable backend code, e.g. `slot_already_booked`. */
  code?: string;
  /** Whether retrying may succeed. Defaults are derived from status/kind. */
  retryable?: boolean;
  /** Correlation id (trace id) for cross-referencing backend logs. */
  traceId?: string;
  /** Structured backend detail, passed through untouched for diagnostics. */
  details?: unknown;
  /** The originating error, when wrapping a transport/parse failure. */
  cause?: unknown;
}

/**
 * The single normalized error every API call rejects with. `instanceof
 * ApiError` is stable across the whole client, so hooks and components branch on
 * `.retryable` / `.status` / `.code` without knowing which layer failed.
 */
export class ApiError extends Error {
  readonly kind: ApiErrorKind;
  readonly status?: number;
  readonly code?: string;
  readonly retryable: boolean;
  readonly traceId?: string;
  readonly details?: unknown;

  constructor(init: ApiErrorInit) {
    super(init.message, { cause: init.cause });
    this.name = 'ApiError';
    this.kind = init.kind;
    this.status = init.status;
    this.code = init.code;
    this.traceId = init.traceId;
    this.details = init.details;
    this.retryable = init.retryable ?? defaultRetryable(init.kind, init.status);
  }

  /** Terminal is simply "not retryable"; exposed for readable call sites. */
  get terminal(): boolean {
    return !this.retryable;
  }
}

/** Retry defaults when the backend does not state `retryable` explicitly. */
function defaultRetryable(kind: ApiErrorKind, status?: number): boolean {
  if (kind === 'network' || kind === 'timeout') return true;
  if (typeof status === 'number') return RETRYABLE_STATUS.has(status);
  return false;
}

function extractField(
  body: BackendErrorBody,
  field: 'message' | 'code',
): string | undefined {
  const nestedError =
    body.error && typeof body.error === 'object' ? body.error : undefined;
  if (field === 'code') {
    return nestedError?.code ?? body.code;
  }
  return (
    nestedError?.message ??
    (typeof body.error === 'string' ? body.error : undefined) ??
    body.message ??
    body.reason ??
    body.detail
  );
}

function extractRetryable(body: BackendErrorBody): boolean | undefined {
  if (body.error && typeof body.error === 'object' && 'retryable' in body.error) {
    return body.error.retryable;
  }
  return body.retryable;
}

/**
 * Normalize a non-2xx REST {@link Response} into an {@link ApiError}. Reads the
 * body once (JSON when possible, else text) and maps conventional fields.
 */
export async function normalizeHttpError(
  response: Response,
  traceId?: string,
): Promise<ApiError> {
  let body: BackendErrorBody | undefined;
  let rawText: string | undefined;
  try {
    rawText = await response.text();
    if (rawText) body = JSON.parse(rawText) as BackendErrorBody;
  } catch {
    // Body was empty or not JSON; fall back to status text / raw text below.
  }

  const message =
    (body && extractField(body, 'message')) ||
    rawText ||
    response.statusText ||
    `Request failed with status ${response.status}`;

  return new ApiError({
    kind: 'http',
    status: response.status,
    code: body && extractField(body, 'code'),
    message,
    retryable: body && extractRetryable(body),
    details: body,
    traceId: traceId ?? response.headers.get('x-request-id') ?? undefined,
  });
}

/** Shape of one GraphQL error entry (subset of the spec we consume). */
export interface GraphQLResponseError {
  message: string;
  extensions?: { code?: string; retryable?: boolean };
}

/** Normalize a GraphQL `errors` array into a single {@link ApiError}. */
export function normalizeGraphQLError(
  errors: GraphQLResponseError[],
  traceId?: string,
): ApiError {
  const [first] = errors;
  return new ApiError({
    kind: 'graphql',
    message: first?.message ?? 'GraphQL request failed',
    code: first?.extensions?.code,
    retryable: first?.extensions?.retryable ?? false,
    details: errors,
    traceId,
  });
}

/**
 * Wrap a thrown transport/timeout/parse failure. Anything already an
 * {@link ApiError} is returned unchanged so we never double-wrap.
 */
export function toApiError(cause: unknown, traceId?: string): ApiError {
  if (cause instanceof ApiError) return cause;

  const isAbort =
    cause instanceof DOMException
      ? cause.name === 'AbortError'
      : cause instanceof Error && cause.name === 'AbortError';

  if (isAbort) {
    return new ApiError({
      kind: 'timeout',
      message: 'The request timed out',
      traceId,
      cause,
    });
  }

  return new ApiError({
    kind: 'network',
    message: cause instanceof Error ? cause.message : 'Network request failed',
    traceId,
    cause,
  });
}

/** Convenience predicate for the TanStack Query `retry` option. */
export function isRetryableError(error: unknown): boolean {
  return error instanceof ApiError && error.retryable;
}
