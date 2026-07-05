'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useRef } from 'react';

import type { Identity } from '@/lib/identity';
import { useUiStore } from '@/store/ui-store';

/**
 * Root client providers.
 *
 * - TanStack Query: a single QueryClient per browser session (created lazily so
 *   it is stable across re-renders and never shared between requests on the
 *   server). Endpoint base URLs live in `@/lib/env` and are consumed by the
 *   query/fetch layer that later stories add.
 * - Zustand: the store is module-level, so instead of a context provider we
 *   hydrate it exactly once with the server-resolved identity. This is the
 *   "Zustand provider" seam — client components read role/user from the store.
 */
export function Providers({
  identity,
  children,
}: {
  identity: Identity;
  children: React.ReactNode;
}) {
  const queryClientRef = useRef<QueryClient>(undefined);
  if (!queryClientRef.current) {
    queryClientRef.current = new QueryClient({
      defaultOptions: {
        queries: {
          staleTime: 30_000,
          refetchOnWindowFocus: false,
          retry: 1,
        },
      },
    });
  }

  // Hydrate the client store with the identity the edge already vouched for.
  // useRef guard keeps this to a single write for the life of the provider.
  const hydratedRef = useRef(false);
  if (!hydratedRef.current) {
    useUiStore.getState().setIdentity({ role: identity.role, user: identity.user });
    hydratedRef.current = true;
  }

  return <QueryClientProvider client={queryClientRef.current}>{children}</QueryClientProvider>;
}
