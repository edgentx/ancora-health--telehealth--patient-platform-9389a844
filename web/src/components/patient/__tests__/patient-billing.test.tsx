import { cleanup, fireEvent, screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it } from 'vitest';

import { server } from '@/test/msw/server';
import { renderWithClient } from '@/test/render';

import { PatientBilling } from '../patient-billing';

afterEach(cleanup);

describe('PatientBilling', () => {
  it('captures a payment against an open invoice', async () => {
    renderWithClient(<PatientBilling />);

    const pay = await screen.findByRole('button', { name: /pay now/i });
    fireEvent.click(pay);

    // Once captured, the invoice flips to paid and the pay button disappears.
    expect(await screen.findByText(/^paid$/i)).toBeTruthy();
    expect(screen.queryByRole('button', { name: /pay now/i })).toBeNull();
  });

  it('surfaces the empty state when there are no invoices', async () => {
    server.use(http.get('*/api/billing/invoices', () => HttpResponse.json([])));
    renderWithClient(<PatientBilling />);
    expect(await screen.findByText(/nothing due/i)).toBeTruthy();
  });
});
