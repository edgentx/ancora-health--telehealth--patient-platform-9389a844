import { fileURLToPath } from 'node:url';

import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

/**
 * Vitest config for the API client unit tests.
 *
 * Tests run under jsdom (React hooks need a DOM), with MSW intercepting the
 * network in a global setup file. The `@` alias mirrors tsconfig so tests import
 * exactly what the app imports. Test files are excluded from the Next build's
 * tsconfig, so this is the only toolchain that compiles them.
 */
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    globals: false,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
  },
});
