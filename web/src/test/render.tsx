/**
 * Shared test render helper.
 *
 * Wraps a component in a fresh TanStack Query client with retries disabled so
 * error-path assertions resolve immediately (no exponential backoff) and each
 * test is fully isolated. Mirrors the wrapper used in the API-client hook tests.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, type RenderOptions } from '@testing-library/react';
import type { ReactElement, ReactNode } from 'react';

export function makeClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

export function renderWithClient(ui: ReactElement, options?: RenderOptions) {
  const client = makeClient();
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  }
  return { client, ...render(ui, { wrapper: Wrapper, ...options }) };
}
