/**
 * Presentation formatting shared by the role surfaces.
 *
 * The backend serializes timestamps as RFC 3339 strings and money as an integer
 * minor unit plus currency (see `@/lib/api` models). These helpers turn those
 * wire shapes into human strings in one place so every view renders them the
 * same way. They are intentionally locale-default and side-effect-free.
 */
import type { Money } from '@/lib/api';

/** Format an ISO date-time as a short, readable local date + time. */
export function formatDateTime(iso: string | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  });
}

/** Format just the time portion (used for slot buttons). */
export function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });
}

/** Format integer-minor-unit {@link Money} using its own currency. */
export function formatMoney(money: Money): string {
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: money.currency,
    }).format(money.amountMinor / 100);
  } catch {
    // Unknown currency code — fall back to a plain decimal + code.
    return `${(money.amountMinor / 100).toFixed(2)} ${money.currency}`;
  }
}

/**
 * Render a metric value with its unit hint. Percentages and plain counts render
 * inline; a `USD` unit is treated as a whole-currency-unit amount (dashboards
 * report revenue in dollars, not cents).
 */
export function formatMetric(value: number, unit?: string): string {
  if (!unit) return value.toLocaleString();
  if (unit === '%') return `${value}%`;
  if (unit === 'USD') {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: 'USD',
      maximumFractionDigits: 0,
    }).format(value);
  }
  return `${value.toLocaleString()} ${unit}`;
}
