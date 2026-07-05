import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useUiStore } from '@/store/ui-store';

import { LogoutButton } from '../logout-button';

// Isolate the edge logout hand-off so the test never touches window.location.
const { beginEdgeLogout } = vi.hoisted(() => ({ beginEdgeLogout: vi.fn() }));

vi.mock('@/lib/auth', () => ({
  beginEdgeLogout,
}));

beforeEach(() => {
  beginEdgeLogout.mockReset();
  // Start signed in as a patient so we can prove logout clears the mirror.
  useUiStore.setState({ role: 'patient', user: 'pat@ancora.test' });
});

afterEach(cleanup);

describe('LogoutButton', () => {
  it('clears the client session mirror and hands off to the edge logout flow', () => {
    render(<LogoutButton />);

    fireEvent.click(screen.getByRole('button', { name: /sign out/i }));

    // Client session state is cleared back to an unauthenticated guest...
    expect(useUiStore.getState().role).toBe('guest');
    expect(useUiStore.getState().user).toBeNull();

    // ...and the browser is sent to the edge logout endpoint, returning to the
    // entry view once the edge ends the session.
    expect(beginEdgeLogout).toHaveBeenCalledTimes(1);
    expect(beginEdgeLogout).toHaveBeenCalledWith('/login');
  });
});
