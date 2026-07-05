import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it } from 'vitest';

import { server } from '@/test/msw/server';
import { renderWithClient } from '@/test/render';

import { SchedulerCalendar } from '../scheduler-calendar';

afterEach(cleanup);

async function loadedSlots(container: HTMLElement) {
  return waitFor(() => {
    const slots = container.querySelectorAll('.slot');
    expect(slots.length).toBeGreaterThan(0);
    return slots;
  });
}

describe('SchedulerCalendar', () => {
  it('books an open slot for a patient', async () => {
    const { container } = renderWithClient(<SchedulerCalendar />);
    await screen.findByRole('option', { name: 'Dr. Amelia Ross' });

    const slots = await loadedSlots(container);
    fireEvent.change(screen.getByLabelText(/patient id/i), {
      target: { value: 'pat-7' },
    });
    fireEvent.click(slots[0] as HTMLElement);

    expect(await screen.findByText(/slot held/i)).toBeTruthy();
  });

  it('surfaces a clear conflict message on a double-book (409)', async () => {
    server.use(
      http.post('*/api/scheduling/appointments/hold', () =>
        HttpResponse.json({ code: 'slot_already_booked' }, { status: 409 }),
      ),
    );

    const { container } = renderWithClient(<SchedulerCalendar />);
    await screen.findByRole('option', { name: 'Dr. Amelia Ross' });
    const slots = await loadedSlots(container);

    fireEvent.click(slots[0] as HTMLElement);

    const alert = await screen.findByRole('alert');
    expect(alert.textContent).toMatch(/just taken/i);
  });

  it('reschedules an existing appointment into a new slot', async () => {
    const { container } = renderWithClient(<SchedulerCalendar />);
    await screen.findByRole('option', { name: 'Dr. Amelia Ross' });
    const slots = await loadedSlots(container);

    // Enter reschedule mode for the existing appointment, then pick a slot.
    fireEvent.click(await screen.findByRole('button', { name: /reschedule/i }));
    expect(await screen.findByText(/choose a new time/i)).toBeTruthy();

    fireEvent.click(slots[1] as HTMLElement);
    expect(await screen.findByText(/appointment updated/i)).toBeTruthy();
  });
});
