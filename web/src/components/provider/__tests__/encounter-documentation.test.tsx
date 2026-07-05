import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it } from 'vitest';

import { server } from '@/test/msw/server';
import { renderWithClient } from '@/test/render';

import { EncounterDocumentation } from '../encounter-documentation';

afterEach(cleanup);

describe('EncounterDocumentation', () => {
  it('documents and signs an encounter, then locks the note', async () => {
    renderWithClient(<EncounterDocumentation />);

    const note = (await screen.findByLabelText(/clinical note/i)) as HTMLTextAreaElement;
    fireEvent.change(note, { target: { value: 'Patient reports improvement.' } });
    fireEvent.click(screen.getByRole('button', { name: /save note/i }));

    fireEvent.click(screen.getByRole('button', { name: /sign encounter/i }));

    // The signed confirmation appears and the note is locked (disabled).
    expect(await screen.findByText(/locked/i)).toBeTruthy();
    await waitFor(() => expect(note.disabled).toBe(true));
  });

  it('places a lab order against the encounter', async () => {
    renderWithClient(<EncounterDocumentation />);

    // Default lab order (CBC) is listed.
    expect(await screen.findByText('CBC')).toBeTruthy();

    const code = screen.getByLabelText(/test code/i) as HTMLInputElement;
    fireEvent.change(code, { target: { value: 'BMP' } });
    fireEvent.click(screen.getByRole('button', { name: /order lab/i }));

    // The composer clears once the order is placed.
    await waitFor(() => expect(code.value).toBe(''));
  });

  it('shows the empty state when there are no encounters', async () => {
    server.use(http.get('*/api/clinical/encounters', () => HttpResponse.json([])));
    renderWithClient(<EncounterDocumentation />);
    expect(await screen.findByText(/no encounters to document/i)).toBeTruthy();
  });
});
