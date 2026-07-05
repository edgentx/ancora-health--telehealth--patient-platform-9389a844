/**
 * Cross-context primitives shared by every bounded-context model.
 *
 * These mirror conventions the Go backend uses on the wire: string identities
 * on every aggregate, RFC 3339 timestamps, integer-minor-unit money, and a
 * uniform pagination envelope. Keeping them here means the per-context models
 * stay focused on their own DTOs.
 */

/** Opaque string identity, matching the backend's `ID string` aggregate ids. */
export type Id = string;

/** RFC 3339 / ISO 8601 timestamp as serialized by Go's `time.Time`. */
export type IsoDateTime = string;

/** Money as an integer minor unit plus currency, avoiding float drift. */
export interface Money {
  /** Amount in the smallest currency unit (e.g. cents). */
  amountMinor: number;
  /** ISO 4217 currency code, e.g. `USD`. */
  currency: string;
}

/** A page of results from a list endpoint. */
export interface Page<T> {
  items: T[];
  /** Opaque cursor for the next page, or `null` when exhausted. */
  nextCursor: string | null;
  /** Total matching records, when the backend computes it. */
  total?: number;
}

/** Common query parameters accepted by list endpoints. */
export interface PageParams {
  cursor?: string;
  limit?: number;
}
