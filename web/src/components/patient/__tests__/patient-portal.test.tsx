import { cleanup, fireEvent, screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it } from 'vitest';

import { server } from '@/test/msw/server';
import { renderWithClient } from '@/test/render';

import { PatientVisits } from '../patient-visits';
import { PatientPrescriptions } from '../patient-prescriptions';
import { IntakeForm } from '../intake-form';

afterEach(cleanup);

describe('PatientVisits', () => {
  it('lists the patient’s visits', async () => {
    renderWithClient(<PatientVisits />);
    expect(await screen.findByText(/booked/i)).toBeTruthy();
  });

  it('shows the empty state when there are no visits', async () => {
    server.use(
      http.get('*/api/scheduling/appointments', () => HttpResponse.json([])),
    );
    renderWithClient(<PatientVisits />);
    expect(await screen.findByText(/no visits scheduled/i)).toBeTruthy();
  });
});

describe('PatientPrescriptions', () => {
  it('lists prescriptions read-only', async () => {
    renderWithClient(<PatientPrescriptions />);
    expect(await screen.findByText(/amoxicillin/i)).toBeTruthy();
  });
});

describe('IntakeForm', () => {
  it('submits intake answers and confirms', async () => {
    renderWithClient(<IntakeForm />);

    fireEvent.change(await screen.findByLabelText(/what brings you in/i), {
      target: { value: 'Sore throat' },
    });
    fireEvent.click(screen.getByRole('button', { name: /submit intake/i }));

    expect(await screen.findByText(/intake submitted/i)).toBeTruthy();
  });
});
