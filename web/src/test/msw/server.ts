/** The shared MSW server used across the API client test suite. */
import { setupServer } from 'msw/node';

import { handlers } from './handlers';

export const server = setupServer(...handlers);
