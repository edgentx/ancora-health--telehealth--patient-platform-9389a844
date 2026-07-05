'use client';

import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

import { beginEdgeLogin } from '@/lib/auth';
import type { Identity } from '@/lib/identity';
import { ROLE_LABELS, ROLE_LANDING } from '@/lib/roles';
import { useUiStore } from '@/store/ui-store';

/**
 * Authentication entry view (`/login`).
 *
 * This is a thin client over the edge-managed login flow. It consumes the
 * identity the Kong+OPA edge already resolved (passed down from the server page
 * that read the trusted headers) and does exactly three things:
 *
 *  1. If the edge vouched for a role, mirror it into the client store and
 *     redirect to that role's home surface.
 *  2. Otherwise, present the unauthenticated entry state whose CTA hands off to
 *     the edge login flow.
 *  3. Surface loading (redirect in flight) and error states around that hand-off.
 *
 * It performs no token validation, password handling, or RBAC — the edge owns
 * all of that. See {@link @/lib/auth} and {@link @/lib/identity}.
 */
export function LoginView({ identity }: { identity: Identity }) {
  const router = useRouter();
  const setIdentity = useUiStore((state) => state.setIdentity);

  const authenticated = identity.role !== 'guest';

  // Keep the client mirror in sync with the edge's answer, then, if the user is
  // authenticated, route them to their surface. Runs after render so navigation
  // never happens during render. The inline `!== 'guest'` check also narrows the
  // role to a concrete surface for the ROLE_LANDING lookup.
  useEffect(() => {
    if (identity.role !== 'guest') {
      setIdentity({ role: identity.role, user: identity.user });
      router.replace(ROLE_LANDING[identity.role]);
    }
  }, [identity.role, identity.user, router, setIdentity]);

  if (authenticated) {
    return <RedirectingState identity={identity} />;
  }

  return <UnauthenticatedState />;
}

/**
 * Shown to an already-authenticated visitor while {@link LoginView} redirects
 * them to their role surface. This is the "authenticated return" state.
 */
function RedirectingState({ identity }: { identity: Identity }) {
  const label = identity.role === 'guest' ? '' : ROLE_LABELS[identity.role];
  return (
    <main className="auth" aria-busy="true">
      <div className="auth__card" role="status" aria-live="polite">
        <p className="auth__brand" style={{ color: 'var(--color-primary)' }}>
          Ancora Health
        </p>
        <div className="auth__spinner" aria-hidden />
        <h1 className="auth__title">Signing you in…</h1>
        <p className="auth__subtitle">Taking you to your {label} workspace.</p>
      </div>
    </main>
  );
}

/**
 * Unauthenticated landing/redirect state. The CTA starts the edge login flow;
 * while the browser navigates away we show a redirecting state, and if the
 * hand-off throws we surface a recoverable error.
 */
function UnauthenticatedState() {
  const [status, setStatus] = useState<'idle' | 'redirecting' | 'error'>('idle');

  function startLogin() {
    setStatus('redirecting');
    try {
      // Hands off to the edge; the browser leaves this page on success.
      beginEdgeLogin(ROLE_LANDING_HINT);
    } catch {
      // Misconfigured edge URL or a blocked navigation — let the user retry.
      setStatus('error');
    }
  }

  return (
    <main className="auth">
      <div className="auth__card">
        <p className="auth__brand" style={{ color: 'var(--color-primary)' }}>
          Ancora Health
        </p>
        <h1 className="auth__title">Sign in to Ancora</h1>
        <p className="auth__subtitle">
          Authentication is handled securely by Ancora&apos;s access edge. Continue to sign in with
          your organization credentials.
        </p>

        {status === 'error' ? (
          <p className="auth__error" role="alert">
            We couldn&apos;t start the sign-in flow. Please try again.
          </p>
        ) : null}

        <button
          type="button"
          className="btn btn--primary auth__cta"
          onClick={startLogin}
          disabled={status === 'redirecting'}
          aria-busy={status === 'redirecting'}
        >
          {status === 'redirecting' ? 'Redirecting…' : 'Continue to secure sign-in'}
        </button>

        <p className="auth__hint">
          You&apos;ll be returned here and routed to your workspace once the edge confirms your
          identity.
        </p>
      </div>
    </main>
  );
}

/**
 * After login the edge drops the user back at the entry view; the server then
 * reads the trusted headers and this view redirects to the right surface. We
 * pass the entry route as the return hint so the round-trip lands here.
 */
const ROLE_LANDING_HINT = '/login';
