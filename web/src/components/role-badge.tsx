'use client';

import { ROLE_LABELS } from '@/lib/roles';
import { useUiStore } from '@/store/ui-store';

/**
 * Reads the resolved role from the hydrated Zustand store and renders it. This
 * is the client-side consumer of the store seam set up in {@link ./providers} —
 * proof that server-resolved identity flows to client components without any
 * token handling on the client.
 */
export function RoleBadge() {
  const role = useUiStore((state) => state.role);
  const user = useUiStore((state) => state.user);
  const label = role === 'guest' ? 'Guest' : ROLE_LABELS[role];

  return (
    <span className="role-badge" title={user ?? undefined}>
      <span className="role-badge__dot" aria-hidden />
      {label}
      {user ? <span className="role-badge__user"> · {user}</span> : null}
    </span>
  );
}
