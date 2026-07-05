import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it } from 'vitest';

import { sampleClinic } from '@/test/msw/handlers';
import { server } from '@/test/msw/server';
import { renderWithClient } from '@/test/render';

import { AdminAnalytics } from '../admin-analytics';

afterEach(cleanup);

describe('AdminAnalytics', () => {
  it('renders utilization, no-show, and revenue metrics from the dashboard', async () => {
    renderWithClient(<AdminAnalytics />);

    expect(await screen.findByText('Utilization')).toBeTruthy();
    expect(screen.getByText('82%')).toBeTruthy();
    expect(screen.getByText('No-show rate')).toBeTruthy();
    expect(screen.getByText('4.1%')).toBeTruthy();
    // Revenue renders as a whole-dollar currency amount.
    expect(screen.getByText(/\$248,000/)).toBeTruthy();
  });

  it('lists clinic directory entries and registers a new clinic', async () => {
    // Stateful directory so registration is reflected on refetch.
    const clinics = [{ ...sampleClinic }];
    server.use(
      http.get('*/api/analytics/clinics', () => HttpResponse.json(clinics)),
      http.post('*/api/analytics/clinics', async ({ request }) => {
        const body = (await request.json()) as { name: string };
        const created = { id: 'clinic-2', name: body.name, active: true, providerIds: [] };
        clinics.push(created);
        return HttpResponse.json(created);
      }),
    );

    renderWithClient(<AdminAnalytics />);
    expect(await screen.findByText('Ancora Downtown')).toBeTruthy();

    const name = screen.getByLabelText(/clinic name/i) as HTMLInputElement;
    fireEvent.change(name, { target: { value: 'Ancora Uptown' } });
    fireEvent.click(screen.getByRole('button', { name: /register clinic/i }));

    expect(await screen.findByText('Ancora Uptown')).toBeTruthy();
    await waitFor(() => expect(name.value).toBe(''));
  });

  it('shows the empty analytics state', async () => {
    server.use(
      http.get('*/api/analytics/dashboards', () => HttpResponse.json([])),
    );
    renderWithClient(<AdminAnalytics />);
    expect(await screen.findByText(/no analytics have been published/i)).toBeTruthy();
  });
});
