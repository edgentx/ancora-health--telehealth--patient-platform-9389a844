import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import { renderWithClient } from '@/test/render';

import { PrescriptionComposer } from '../prescription-composer';

afterEach(cleanup);

describe('PrescriptionComposer', () => {
  it('drafts a prescription and confirms it', async () => {
    renderWithClient(<PrescriptionComposer />);

    // Existing prescriptions load in the side panel.
    expect(await screen.findByText(/amoxicillin/i)).toBeTruthy();

    fireEvent.change(screen.getByLabelText(/patient id/i), {
      target: { value: 'pat-9' },
    });
    fireEvent.change(screen.getByLabelText(/medication/i), {
      target: { value: 'Lisinopril 10mg' },
    });
    fireEvent.change(screen.getByLabelText(/dosage/i), {
      target: { value: '1 tablet daily' },
    });

    const submit = screen.getByRole('button', { name: /compose prescription/i });
    fireEvent.click(submit);

    expect(await screen.findByText(/prescription drafted/i)).toBeTruthy();
    // The medication field resets after a successful compose.
    await waitFor(() =>
      expect((screen.getByLabelText(/medication/i) as HTMLInputElement).value).toBe(''),
    );
  });

  it('keeps submit disabled until every field is filled', async () => {
    renderWithClient(<PrescriptionComposer />);
    await screen.findByText(/amoxicillin/i);

    const submit = screen.getByRole('button', {
      name: /compose prescription/i,
    }) as HTMLButtonElement;
    expect(submit.disabled).toBe(true);
  });
});
