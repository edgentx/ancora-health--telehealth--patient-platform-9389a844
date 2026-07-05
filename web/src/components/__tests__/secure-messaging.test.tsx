import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { graphql, http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it } from 'vitest';

import { sampleMessage } from '@/test/msw/handlers';
import { server } from '@/test/msw/server';
import { renderWithClient } from '@/test/render';

import { SecureMessaging } from '../secure-messaging';

afterEach(cleanup);

describe('SecureMessaging', () => {
  it('shows a thread, its messages, and posts a reply that renders as mine', async () => {
    // Stateful thread so the post → invalidate → refetch cycle actually grows the list.
    const messages = [{ ...sampleMessage }];
    server.use(
      graphql.query('ThreadMessages', () =>
        HttpResponse.json({ data: { threadMessages: messages } }),
      ),
      http.post('*/api/engagement/threads/:id/messages', async ({ request }) => {
        const body = (await request.json()) as { body: string };
        const created = {
          id: 'msg-2',
          threadId: 'thread-1',
          authorId: 'pat-1',
          body: body.body,
          sentAt: '2026-07-06T16:00:00Z',
        };
        messages.push(created);
        return HttpResponse.json(created);
      }),
    );

    renderWithClient(<SecureMessaging />);

    // Thread + its first message load from the mocked GraphQL surface.
    expect(await screen.findByText('How are you feeling?')).toBeTruthy();

    // Compose and send a reply.
    const input = screen.getByLabelText('Message') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'Much better, thanks' } });
    fireEvent.click(screen.getByRole('button', { name: /^send$/i }));

    // The reply appears and is styled as our own message; the composer clears.
    const reply = await screen.findByText('Much better, thanks');
    expect(reply.closest('.message')?.className).toMatch(/message--mine/);
    await waitFor(() => expect(input.value).toBe(''));
  });

  it('renders the empty state when there are no threads', async () => {
    server.use(
      graphql.query('Threads', () =>
        HttpResponse.json({ data: { messageThreads: [] } }),
      ),
    );
    renderWithClient(<SecureMessaging />);
    expect(await screen.findByText(/no secure messages yet/i)).toBeTruthy();
  });
});
