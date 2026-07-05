'use client';

import { beginEdgeLogout } from '@/lib/auth';
import { useUiStore } from '@/store/ui-store';

/**
 * Logout control shown in the app-shell topbar.
 *
 * Logout is a two-step hand-off: first clear the client's local identity mirror
 * so no stale role lingers in the UI, then navigate to the edge logout endpoint,
 * which invalidates the session and returns the user to the entry view as a
 * `guest`. The client never ends the session on its own — it defers to the edge.
 */
export function LogoutButton() {
  const clearIdentity = useUiStore((state) => state.clearIdentity);

  function handleLogout() {
    clearIdentity();
    // Return to the entry view; the edge ends the session and lands here.
    beginEdgeLogout('/login');
  }

  return (
    <button type="button" className="btn btn--ghost logout-btn" onClick={handleLogout}>
      Sign out
    </button>
  );
}
