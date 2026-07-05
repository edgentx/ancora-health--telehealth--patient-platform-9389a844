/**
 * Typed REST transport.
 *
 * A thin wrapper over `fetch` that centralizes everything a call site should
 * never repeat: the edge base URL, JSON headers, credentialed cookies, W3C
 * trace propagation, a per-request timeout, retry-with-backoff on transient
 * failures, and normalization of every outcome into an {@link ApiError}. It is
 * transport-only — bounded-context semantics live in the model/hook layers.
 */

import { publicEnv } from '@/lib/env';

import { ApiError, normalizeHttpError, toApiError } from './errors';
import { newTraceContext } from './trace';

/** Immutable client configuration shared by REST and GraphQL. */
export interface HttpClientConfig {
  /** Edge base URL, e.g. `https://api.ancora.vforce360.ai`. */
  baseUrl: string;
  /** Headers merged into every request (e.g. an API key or client version). */
  defaultHeaders: Record<string, string>;
  /** `fetch` credentials mode; `include` so the edge sees the session cookie. */
  credentials: RequestCredentials;
  /** Per-request timeout in ms before the request is aborted as a timeout. */
  timeoutMs: number;
  /** Max retry attempts for transient failures (0 disables retries). */
  maxRetries: number;
  /** Base backoff delay in ms; grows exponentially with jitter per attempt. */
  retryBackoffMs: number;
}

/** Per-call options layered on top of the shared config. */
export interface RequestOptions {
  method?: string;
  /** Query-string params; `undefined`/`null` values are dropped. */
  query?: Record<string, string | number | boolean | undefined | null>;
  /** JSON-serialized as the request body (sets `content-type`). */
  body?: unknown;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  /** Override the config timeout for this call. */
  timeoutMs?: number;
  /**
   * Force retry eligibility. By default only idempotent methods (GET/HEAD/
   * PUT/DELETE) are retried; set `true` to retry a POST you know is safe.
   */
  retry?: boolean;
}

const IDEMPOTENT_METHODS = new Set(['GET', 'HEAD', 'PUT', 'DELETE', 'OPTIONS']);

/** The default configuration derived from the browser-visible environment. */
export function defaultConfig(): HttpClientConfig {
  return {
    baseUrl: publicEnv.apiBaseUrl,
    defaultHeaders: { accept: 'application/json' },
    credentials: 'include',
    timeoutMs: 15_000,
    maxRetries: 2,
    retryBackoffMs: 300,
  };
}

function buildUrl(
  baseUrl: string,
  path: string,
  query?: RequestOptions['query'],
): string {
  const url = new URL(
    path.replace(/^\//, ''),
    baseUrl.endsWith('/') ? baseUrl : `${baseUrl}/`,
  );
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== null) {
        url.searchParams.set(key, String(value));
      }
    }
  }
  return url.toString();
}

/** Exponential backoff with full jitter; honours a `Retry-After` hint. */
function backoffDelay(
  attempt: number,
  baseMs: number,
  retryAfter?: string | null,
): number {
  if (retryAfter) {
    const seconds = Number(retryAfter);
    if (Number.isFinite(seconds) && seconds >= 0) return seconds * 1000;
  }
  const ceiling = baseMs * 2 ** attempt;
  return Math.round(Math.random() * ceiling);
}

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException('Aborted', 'AbortError'));
      return;
    }
    const timer = setTimeout(resolve, ms);
    signal?.addEventListener(
      'abort',
      () => {
        clearTimeout(timer);
        reject(new DOMException('Aborted', 'AbortError'));
      },
      { once: true },
    );
  });
}

/**
 * The REST client. One instance is shared across the app (see `./client`),
 * exposing typed verb helpers that all funnel through {@link request}.
 */
export class HttpClient {
  constructor(readonly config: HttpClientConfig) {}

  /** Issue a request and decode a JSON body of type `T` (or `void` on 204). */
  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const method = (options.method ?? 'GET').toUpperCase();
    const url = buildUrl(this.config.baseUrl, path, options.query);
    const trace = newTraceContext();

    const retryEligible =
      options.retry ?? IDEMPOTENT_METHODS.has(method);
    const maxAttempts = retryEligible ? this.config.maxRetries + 1 : 1;

    let lastError: ApiError | undefined;

    for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
      try {
        const response = await this.fetchOnce(url, method, trace.traceId, options);

        if (response.ok) {
          return await decodeBody<T>(response, trace.traceId);
        }

        const error = await normalizeHttpError(response, trace.traceId);
        // Retry transient server-side failures; surface terminal ones at once.
        if (error.retryable && attempt < maxAttempts - 1) {
          lastError = error;
          await sleep(
            backoffDelay(
              attempt,
              this.config.retryBackoffMs,
              response.headers.get('retry-after'),
            ),
            options.signal,
          );
          continue;
        }
        throw error;
      } catch (caught) {
        const error = toApiError(caught, trace.traceId);
        // A caller-driven abort is not a retry candidate.
        if (options.signal?.aborted) throw error;
        if (error.retryable && attempt < maxAttempts - 1) {
          lastError = error;
          await sleep(backoffDelay(attempt, this.config.retryBackoffMs), options.signal);
          continue;
        }
        throw error;
      }
    }

    // Unreachable in practice: the loop always returns or throws. Guard anyway.
    throw (
      lastError ??
      new ApiError({ kind: 'unknown', message: 'Request failed', traceId: trace.traceId })
    );
  }

  private async fetchOnce(
    url: string,
    method: string,
    traceId: string,
    options: RequestOptions,
  ): Promise<Response> {
    const controller = new AbortController();
    const timeoutMs = options.timeoutMs ?? this.config.timeoutMs;
    const timer = setTimeout(() => controller.abort(), timeoutMs);

    // Fresh span per attempt, but the trace id is stable across the retry loop.
    const trace = newTraceContext(traceId);

    // Chain a caller-supplied signal into our timeout controller.
    if (options.signal) {
      if (options.signal.aborted) controller.abort();
      else
        options.signal.addEventListener('abort', () => controller.abort(), {
          once: true,
        });
    }

    const headers: Record<string, string> = {
      ...this.config.defaultHeaders,
      ...trace.headers,
      ...options.headers,
    };

    let bodyInit: BodyInit | undefined;
    if (options.body !== undefined) {
      headers['content-type'] = headers['content-type'] ?? 'application/json';
      bodyInit = JSON.stringify(options.body);
    }

    try {
      return await fetch(url, {
        method,
        headers,
        body: bodyInit,
        credentials: this.config.credentials,
        signal: controller.signal,
      });
    } finally {
      clearTimeout(timer);
    }
  }

  get<T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
    return this.request<T>(path, { ...options, method: 'GET' });
  }

  post<T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method'>): Promise<T> {
    return this.request<T>(path, { ...options, method: 'POST', body });
  }

  put<T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method'>): Promise<T> {
    return this.request<T>(path, { ...options, method: 'PUT', body });
  }

  patch<T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method'>): Promise<T> {
    return this.request<T>(path, { ...options, method: 'PATCH', body });
  }

  delete<T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
    return this.request<T>(path, { ...options, method: 'DELETE' });
  }
}

/** Decode a successful response body, tolerating empty (204/205) responses. */
async function decodeBody<T>(response: Response, traceId: string): Promise<T> {
  if (response.status === 204 || response.status === 205) {
    return undefined as T;
  }
  const text = await response.text();
  if (!text) return undefined as T;
  try {
    return JSON.parse(text) as T;
  } catch (cause) {
    throw new ApiError({
      kind: 'parse',
      message: 'Failed to decode response body as JSON',
      status: response.status,
      traceId,
      cause,
    });
  }
}
