/**
 * Global test setup: start MSW before the suite, reset handlers between tests
 * (so `server.use` overrides never leak), and stop it at the end. `onUnhandled`
 * errors so a forgotten mock surfaces as a failure rather than a real fetch.
 */
import { afterAll, afterEach, beforeAll } from 'vitest';

import { server } from './msw/server';

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
