/**
 * Default MSW request handlers.
 *
 * These provide happy-path responses for the endpoints the client calls so the
 * bulk of tests need no per-test setup. Individual specs override a route with
 * `server.use(...)` to exercise error and retry paths.
 */
import { graphql, http, HttpResponse } from 'msw';

import type { Appointment } from '@/lib/api/models/scheduling';
import type { Message, MessageThread } from '@/lib/api/models/engagement';

export const sampleAppointment: Appointment = {
  id: 'appt-1',
  status: 'booked',
  providerId: 'prov-1',
  patientId: 'pat-1',
  timeSlot: '2026-07-06T15:00:00Z',
  version: 2,
};

export const sampleThread: MessageThread = {
  id: 'thread-1',
  status: 'open',
  patientId: 'pat-1',
  careTeamMemberIds: ['prov-1'],
  subject: 'Follow-up',
  version: 1,
};

export const sampleMessage: Message = {
  id: 'msg-1',
  threadId: 'thread-1',
  authorId: 'prov-1',
  body: 'How are you feeling?',
  sentAt: '2026-07-06T15:05:00Z',
};

export const handlers = [
  http.get('*/api/scheduling/appointments', () =>
    HttpResponse.json([sampleAppointment]),
  ),
  http.post('*/api/scheduling/appointments/hold', () =>
    HttpResponse.json(sampleAppointment),
  ),

  graphql.query('Threads', () =>
    HttpResponse.json({ data: { messageThreads: [sampleThread] } }),
  ),
  graphql.query('ThreadMessages', () =>
    HttpResponse.json({ data: { threadMessages: [sampleMessage] } }),
  ),
];
