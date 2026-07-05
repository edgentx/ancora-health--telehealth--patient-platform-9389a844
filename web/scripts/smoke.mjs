/**
 * Production smoke test.
 *
 * Builds are verified by `next build`; this script proves the running app
 * actually renders each role's landing route when the edge supplies the trusted
 * role header. It boots `next start`, then requests every landing route with the
 * matching `x-ancora-role` header and asserts the role's workspace renders
 * (not the role gate). Run after `npm run build`:  `npm run smoke`.
 */
import { spawn } from 'node:child_process';

const PORT = process.env.SMOKE_PORT ?? '4123';
const BASE = `http://127.0.0.1:${PORT}`;
const ROLE_HEADER = process.env.ANCORA_ROLE_HEADER ?? 'x-ancora-role';
const USER_HEADER = process.env.ANCORA_USER_HEADER ?? 'x-ancora-user';

// role -> { landing route, a contiguous heading unique to that role's landing }.
// (Avoid strings that span a JSX interpolation — React's SSR inserts a
// `<!-- -->` comment at the boundary, which would break a substring match.)
const CASES = [
  { role: 'patient', path: '/dashboard', expect: 'Your health, at a glance' },
  { role: 'provider', path: '/provider/dashboard', expect: 'Today at a glance' },
  { role: 'scheduler', path: '/scheduler/dashboard', expect: 'Front desk' },
  { role: 'admin', path: '/admin/dashboard', expect: 'Clinic overview' },
];

// Text rendered only by the role gate; its absence proves the surface rendered.
const GATE_MARKER = 'Not available for your role';

function log(msg) {
  process.stdout.write(`[smoke] ${msg}\n`);
}

async function waitForReady(timeoutMs = 30_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(BASE, { method: 'GET' });
      if (res.ok || res.status === 200) return;
    } catch {
      // server not up yet
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`server did not become ready within ${timeoutMs}ms`);
}

async function checkCase({ role, path, expect }) {
  const res = await fetch(`${BASE}${path}`, {
    headers: { [ROLE_HEADER]: role, [USER_HEADER]: `${role}@smoke.test` },
  });
  const body = await res.text();
  if (res.status !== 200) {
    throw new Error(`${role} ${path}: expected 200, got ${res.status}`);
  }
  if (body.includes(GATE_MARKER)) {
    throw new Error(`${role} ${path}: role gate rendered instead of the surface`);
  }
  if (!body.includes(expect)) {
    throw new Error(`${role} ${path}: rendered output missing "${expect}"`);
  }
  log(`ok  ${role.padEnd(9)} ${path} -> 200, surface rendered ("${expect}")`);
}

async function main() {
  log(`starting "next start" on :${PORT}`);
  const server = spawn('npx', ['next', 'start', '-p', PORT], {
    stdio: ['ignore', 'inherit', 'inherit'],
    env: process.env,
  });

  let exitCode = 0;
  try {
    await waitForReady();
    for (const c of CASES) {
      await checkCase(c);
    }
    log('all role landing routes rendered ✓');
  } catch (err) {
    exitCode = 1;
    process.stderr.write(`[smoke] FAILED: ${err.message}\n`);
  } finally {
    server.kill('SIGTERM');
  }
  process.exit(exitCode);
}

main();
