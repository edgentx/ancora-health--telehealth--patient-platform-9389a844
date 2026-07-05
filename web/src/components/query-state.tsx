'use client';

import type { ReactNode } from 'react';

import { ApiError } from '@/lib/api';

/**
 * The subset of a TanStack Query result this boundary needs. Accepting a
 * structural type (rather than `UseQueryResult<T>`) keeps the component trivial
 * to drive from tests and reusable across every bounded-context hook.
 */
export interface QueryLike<T> {
  data: T | undefined;
  isPending: boolean;
  isError: boolean;
  error: unknown;
  refetch?: () => void;
}

/**
 * One place that renders the loading / error / empty / success states every
 * role surface needs, so no view re-implements the four-way branch. The error
 * state reads the normalized {@link ApiError} for a human message and offers a
 * retry affordance for transient failures.
 */
export function QueryState<T>({
  query,
  isEmpty,
  loadingLabel = 'Loading…',
  emptyLabel = 'Nothing to show yet.',
  children,
}: {
  query: QueryLike<T>;
  /** Predicate marking a successful-but-empty result (e.g. `d.length === 0`). */
  isEmpty?: (data: T) => boolean;
  loadingLabel?: string;
  emptyLabel?: ReactNode;
  children: (data: T) => ReactNode;
}) {
  if (query.isPending) {
    return (
      <div className="qs qs--loading" role="status" aria-live="polite">
        <span className="qs__spinner" aria-hidden />
        {loadingLabel}
      </div>
    );
  }

  if (query.isError) {
    return <QueryError error={query.error} onRetry={query.refetch} />;
  }

  const data = query.data as T;
  if (isEmpty?.(data)) {
    return (
      <div className="qs qs--empty" role="status">
        {emptyLabel}
      </div>
    );
  }

  return <>{children(data)}</>;
}

/** Human-facing error card derived from the normalized {@link ApiError}. */
export function QueryError({
  error,
  onRetry,
}: {
  error: unknown;
  onRetry?: () => void;
}) {
  const message =
    error instanceof ApiError
      ? error.message
      : error instanceof Error
        ? error.message
        : 'Something went wrong.';
  const retryable = !(error instanceof ApiError) || error.retryable;

  return (
    <div className="qs qs--error" role="alert">
      <p className="qs__error-title">We couldn’t load this.</p>
      <p className="qs__error-body">{message}</p>
      {retryable && onRetry ? (
        <button type="button" className="btn btn--ghost" onClick={() => onRetry()}>
          Try again
        </button>
      ) : null}
    </div>
  );
}
