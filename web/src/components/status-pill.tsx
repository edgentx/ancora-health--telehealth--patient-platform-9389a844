import type { ReactNode } from 'react';

/**
 * A lifecycle-status pill with a tone derived from the status string. Aggregate
 * statuses across contexts share vocabulary (booked/paid/signed = good,
 * cancelled/failed/void = bad, held/pending/draft = in-progress), so one map
 * keeps their colouring consistent everywhere a status is shown.
 */
const TONE: Record<string, 'ok' | 'warn' | 'danger' | 'muted'> = {
  // good / terminal-success
  booked: 'ok',
  paid: 'ok',
  captured: 'ok',
  signed: 'ok',
  issued: 'ok',
  resulted: 'ok',
  submitted: 'ok',
  active: 'ok',
  // in-progress / attention
  held: 'warn',
  pending: 'warn',
  draft: 'warn',
  drafted: 'warn',
  open: 'warn',
  ordered: 'warn',
  collected: 'warn',
  safety_checked: 'warn',
  amended: 'warn',
  // terminal-negative
  cancelled: 'danger',
  failed: 'danger',
  void: 'danger',
  inactive: 'danger',
  // neutral
  new: 'muted',
};

export function StatusPill({
  status,
  children,
}: {
  status: string;
  children?: ReactNode;
}) {
  const tone = TONE[status] ?? 'muted';
  const label = children ?? status.replace(/_/g, ' ');
  return <span className={`pill pill--${tone}`}>{label}</span>;
}
