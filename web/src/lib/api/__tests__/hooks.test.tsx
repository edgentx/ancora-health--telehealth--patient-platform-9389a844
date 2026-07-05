import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { createElement, type ReactNode } from 'react';
import { describe, expect, it } from 'vitest';

import { server } from '@/test/msw/server';

import { sampleAppointment } from '@/test/msw/handlers';
import { useAppointments, useHoldSlot } from '../hooks/scheduling';
import { useMessageThreads } from '../hooks/engagement';

function wrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
}

describe('scheduling hooks', () => {
  it('reads appointments over REST with the mocked network', async () => {
    const { result } = renderHook(() => useAppointments(), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([sampleAppointment]);
  });

  it('invalidates the appointments query after a hold mutation', async () => {
    let getCalls = 0;
    server.use(
      http.get('*/api/scheduling/appointments', () => {
        getCalls += 1;
        return HttpResponse.json([sampleAppointment]);
      }),
      http.post('*/api/scheduling/appointments/hold', () =>
        HttpResponse.json(sampleAppointment),
      ),
    );

    const { result } = renderHook(
      () => ({ list: useAppointments(), hold: useHoldSlot() }),
      { wrapper: wrapper() },
    );

    await waitFor(() => expect(result.current.list.isSuccess).toBe(true));
    expect(getCalls).toBe(1);

    result.current.hold.mutate({
      providerId: 'prov-1',
      patientId: 'pat-1',
      timeSlot: '2026-07-06T15:00:00Z',
    });

    // onSuccess invalidates the appointments key → a second GET fires.
    await waitFor(() => expect(getCalls).toBe(2));
  });
});

describe('engagement hooks', () => {
  it('reads message threads over GraphQL', async () => {
    const { result } = renderHook(() => useMessageThreads(), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0]?.subject).toBe('Follow-up');
  });
});
