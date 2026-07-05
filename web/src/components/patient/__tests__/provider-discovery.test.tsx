import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it } from 'vitest';

import { server } from '@/test/msw/server';
import { renderWithClient } from '@/test/render';

import { ProviderDiscovery } from '../provider-discovery';

afterEach(cleanup);

describe('ProviderDiscovery', () => {
  it('lists providers and books an open slot', async () => {
    const { container } = renderWithClient(<ProviderDiscovery />);

    // Directory loads from the mocked network.
    await screen.findAllByText('Dr. Amelia Ross');

    // Slots for the (auto-selected) first provider render...
    const slots = await waitFor(() => {
      const found = container.querySelectorAll('.slot');
      expect(found.length).toBeGreaterThan(0);
      return found;
    });

    // ...and holding one surfaces the booking confirmation.
    fireEvent.click(slots[0] as HTMLElement);
    expect(await screen.findByText(/Booked/i)).toBeTruthy();
  });

  it('shows a conflict message when the slot was just taken (409)', async () => {
    server.use(
      http.post('*/api/scheduling/appointments/hold', () =>
        HttpResponse.json(
          { code: 'slot_already_booked', message: 'taken' },
          { status: 409 },
        ),
      ),
    );

    const { container } = renderWithClient(<ProviderDiscovery />);
    await screen.findAllByText('Dr. Amelia Ross');
    const slots = await waitFor(() => {
      const found = container.querySelectorAll('.slot');
      expect(found.length).toBeGreaterThan(0);
      return found;
    });

    fireEvent.click(slots[0] as HTMLElement);
    const alert = await screen.findByRole('alert');
    expect(alert.textContent).toMatch(/just taken/i);
  });

  it('renders the empty state when no providers are available', async () => {
    server.use(
      http.get('*/api/scheduling/providers', () => HttpResponse.json([])),
    );
    renderWithClient(<ProviderDiscovery />);
    expect(await screen.findByText(/no providers are accepting/i)).toBeTruthy();
  });
});
