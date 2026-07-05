import { create } from 'zustand';

import type { ResolvedRole } from '@/lib/roles';

/**
 * Client-side UI + identity store.
 *
 * The role/user here is a *mirror* of what the edge already resolved server-side
 * (hydrated once by the root providers). It exists so client components can gate
 * presentation without re-reading headers — it is NOT an authorization source.
 */
interface UiState {
  role: ResolvedRole;
  user: string | null;
  sidebarOpen: boolean;
  setIdentity: (identity: { role: ResolvedRole; user: string | null }) => void;
  /**
   * Reset the local identity mirror to unauthenticated `guest`. Used by logout
   * to clear client session state *before* handing off to the edge — the edge
   * remains the authority that actually ends the session.
   */
  clearIdentity: () => void;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
}

export const useUiStore = create<UiState>((set) => ({
  role: 'guest',
  user: null,
  sidebarOpen: true,
  setIdentity: (identity) => set({ role: identity.role, user: identity.user }),
  clearIdentity: () => set({ role: 'guest', user: null }),
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
}));
