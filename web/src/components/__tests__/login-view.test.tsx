import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ROLE_LANDING, ROLES } from '@/lib/roles';
import { useUiStore } from '@/store/ui-store';

import { LoginView } from '../login-view';

// Isolate the two side-effecting seams: the router (authenticated redirect) and
// the edge login hand-off. Hoisted so the mock fns exist before module load.
const { replace, beginEdgeLogin } = vi.hoisted(() => ({
  replace: vi.fn(),
  beginEdgeLogin: vi.fn(),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace }),
}));

vi.mock('@/lib/auth', () => ({
  beginEdgeLogin,
}));

beforeEach(() => {
  replace.mockReset();
  beginEdgeLogin.mockReset();
  // Reset the shared module-level store to unauthenticated between tests.
  useUiStore.setState({ role: 'guest', user: null });
});

afterEach(cleanup);

describe('LoginView — authenticated return', () => {
  // Acceptance: authenticated users are redirected to their role's home surface.
  it.each(ROLES)('redirects a %s to their landing surface and mirrors identity', (role) => {
    render(<LoginView identity={{ role, user: `${role}@ancora.test` }} />);

    // Redirected to the role's home surface...
    expect(replace).toHaveBeenCalledTimes(1);
    expect(replace).toHaveBeenCalledWith(ROLE_LANDING[role]);

    // ...the client mirror is synced to the edge-resolved identity...
    expect(useUiStore.getState().role).toBe(role);
    expect(useUiStore.getState().user).toBe(`${role}@ancora.test`);

    // ...and the user sees the "signing in" hand-off, not the sign-in CTA.
    expect(screen.getByRole('status').textContent).toMatch(/signing you in/i);
    expect(screen.queryByRole('button', { name: /secure sign-in/i })).toBeNull();
  });
});

describe('LoginView — unauthenticated entry', () => {
  it('shows the entry state and never redirects a guest', () => {
    render(<LoginView identity={{ role: 'guest', user: null }} />);

    expect(screen.getByRole('heading', { name: /sign in to ancora/i })).toBeTruthy();
    expect(screen.getByRole('button', { name: /continue to secure sign-in/i })).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });

  it('triggers the edge login flow when the CTA is clicked', () => {
    render(<LoginView identity={{ role: 'guest', user: null }} />);

    fireEvent.click(screen.getByRole('button', { name: /continue to secure sign-in/i }));

    expect(beginEdgeLogin).toHaveBeenCalledTimes(1);
    // Button flips to its in-flight (loading) state while the browser navigates.
    const redirecting = screen.getByRole('button', { name: /redirecting/i }) as HTMLButtonElement;
    expect(redirecting.disabled).toBe(true);
  });

  it('surfaces a recoverable error when the login hand-off throws', () => {
    beginEdgeLogin.mockImplementation(() => {
      throw new Error('edge unreachable');
    });
    render(<LoginView identity={{ role: 'guest', user: null }} />);

    fireEvent.click(screen.getByRole('button', { name: /continue to secure sign-in/i }));

    expect(screen.getByRole('alert').textContent).toMatch(/couldn.t start the sign-in flow/i);
    // Still recoverable: the CTA is enabled again for a retry.
    const cta = screen.getByRole('button', {
      name: /continue to secure sign-in/i,
    }) as HTMLButtonElement;
    expect(cta.disabled).toBe(false);
  });
});
