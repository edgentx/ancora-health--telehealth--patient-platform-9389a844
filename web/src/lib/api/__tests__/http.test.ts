import { delay, http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { server } from '@/test/msw/server';

import { ApiError } from '../errors';
import { HttpClient, defaultConfig } from '../http';

const BASE = 'http://api.test';

function makeClient(overrides: Partial<ReturnType<typeof defaultConfig>> = {}) {
  return new HttpClient({
    ...defaultConfig(),
    baseUrl: BASE,
    retryBackoffMs: 1,
    ...overrides,
  });
}

describe('HttpClient', () => {
  it('decodes a successful JSON response', async () => {
    server.use(
      http.get(`${BASE}/api/thing`, () => HttpResponse.json({ ok: true })),
    );
    const client = makeClient();
    await expect(client.get('/api/thing')).resolves.toEqual({ ok: true });
  });

  it('propagates W3C trace headers on the outbound request', async () => {
    let traceparent: string | null = null;
    let requestId: string | null = null;
    server.use(
      http.get(`${BASE}/api/thing`, ({ request }) => {
        traceparent = request.headers.get('traceparent');
        requestId = request.headers.get('x-request-id');
        return HttpResponse.json({});
      }),
    );
    await makeClient().get('/api/thing');
    expect(traceparent).toMatch(/^00-[0-9a-f]{32}-[0-9a-f]{16}-01$/);
    expect(requestId).toMatch(/^[0-9a-f]{32}$/);
  });

  it('normalizes a terminal 4xx into a non-retryable ApiError', async () => {
    server.use(
      http.get(`${BASE}/api/thing`, () =>
        HttpResponse.json(
          { error: { code: 'slot_already_booked', message: 'Slot taken' } },
          { status: 409 },
        ),
      ),
    );
    const error = await makeClient()
      .get('/api/thing')
      .catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(409);
    expect((error as ApiError).code).toBe('slot_already_booked');
    expect((error as ApiError).message).toBe('Slot taken');
    expect((error as ApiError).retryable).toBe(false);
    expect((error as ApiError).terminal).toBe(true);
  });

  it('retries a transient 503 and then succeeds', async () => {
    let calls = 0;
    server.use(
      http.get(`${BASE}/api/thing`, () => {
        calls += 1;
        if (calls < 3) return new HttpResponse(null, { status: 503 });
        return HttpResponse.json({ recovered: true });
      }),
    );
    const client = makeClient({ maxRetries: 3 });
    await expect(client.get('/api/thing')).resolves.toEqual({ recovered: true });
    expect(calls).toBe(3);
  });

  it('stops retrying once attempts are exhausted', async () => {
    let calls = 0;
    server.use(
      http.get(`${BASE}/api/thing`, () => {
        calls += 1;
        return new HttpResponse(null, { status: 503 });
      }),
    );
    const client = makeClient({ maxRetries: 2 });
    const error = await client.get('/api/thing').catch((e: unknown) => e);
    expect((error as ApiError).status).toBe(503);
    expect((error as ApiError).retryable).toBe(true);
    expect(calls).toBe(3); // initial + 2 retries
  });

  it('does not retry non-idempotent POST by default', async () => {
    let calls = 0;
    server.use(
      http.post(`${BASE}/api/thing`, () => {
        calls += 1;
        return new HttpResponse(null, { status: 503 });
      }),
    );
    await makeClient({ maxRetries: 3 })
      .post('/api/thing', { a: 1 })
      .catch(() => undefined);
    expect(calls).toBe(1);
  });

  it('surfaces a timeout as a retryable ApiError of kind timeout', async () => {
    server.use(
      http.get(`${BASE}/api/slow`, async () => {
        await delay(100);
        return HttpResponse.json({});
      }),
    );
    const client = makeClient({ timeoutMs: 20, maxRetries: 0 });
    const error = await client.get('/api/slow').catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).kind).toBe('timeout');
    expect((error as ApiError).retryable).toBe(true);
  });
});
