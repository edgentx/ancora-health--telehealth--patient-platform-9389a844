import { describe, expect, it } from 'vitest';

import {
  ApiError,
  isRetryableError,
  normalizeGraphQLError,
  normalizeHttpError,
  toApiError,
} from '../errors';

describe('error normalization', () => {
  it('classifies transient statuses as retryable', async () => {
    const error = await normalizeHttpError(
      new Response('{}', { status: 503 }),
      'trace-1',
    );
    expect(error.retryable).toBe(true);
    expect(error.traceId).toBe('trace-1');
  });

  it('classifies client errors as terminal', async () => {
    const error = await normalizeHttpError(
      new Response(JSON.stringify({ message: 'bad input' }), { status: 400 }),
    );
    expect(error.retryable).toBe(false);
    expect(error.message).toBe('bad input');
  });

  it('honours an explicit backend retryable flag over the status default', async () => {
    const error = await normalizeHttpError(
      new Response(JSON.stringify({ message: 'locked', retryable: true }), {
        status: 423,
      }),
    );
    expect(error.retryable).toBe(true);
  });

  it('wraps an AbortError as a timeout', () => {
    const error = toApiError(new DOMException('Aborted', 'AbortError'));
    expect(error.kind).toBe('timeout');
    expect(error.retryable).toBe(true);
  });

  it('wraps a generic transport failure as a retryable network error', () => {
    const error = toApiError(new TypeError('Failed to fetch'));
    expect(error.kind).toBe('network');
    expect(error.retryable).toBe(true);
  });

  it('never double-wraps an existing ApiError', () => {
    const original = new ApiError({ kind: 'http', message: 'x', status: 404 });
    expect(toApiError(original)).toBe(original);
  });

  it('normalizes GraphQL errors and exposes a retry predicate', () => {
    const error = normalizeGraphQLError([{ message: 'nope' }]);
    expect(error.kind).toBe('graphql');
    expect(isRetryableError(error)).toBe(false);
    expect(isRetryableError(new ApiError({ kind: 'network', message: 'x' }))).toBe(true);
  });
});
