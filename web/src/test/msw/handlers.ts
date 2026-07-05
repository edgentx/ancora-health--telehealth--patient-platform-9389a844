/**
 * Default MSW request handlers.
 *
 * These provide happy-path responses for every endpoint the S-80 client calls,
 * so the bulk of component/hook tests need no per-test setup. Individual specs
 * override a route with `server.use(...)` to exercise error, empty, and conflict
 * paths (e.g. a 409 double-book on reschedule). Exported sample fixtures keep the
 * shapes in one place so assertions stay in lockstep with the mocks.
 */
import { graphql, http, HttpResponse } from 'msw';

import type {
  Appointment,
  ProviderSchedule,
  ProviderSummary,
} from '@/lib/api/models/scheduling';
import type { Encounter, LabOrder } from '@/lib/api/models/clinical';
import type {
  IntakeAnswer,
  IntakeForm,
  Message,
  MessageThread,
  Notification,
  Prescription,
} from '@/lib/api/models/engagement';
import type { Invoice, Payment } from '@/lib/api/models/billing';
import type {
  AnalyticsDashboard,
  ClinicDirectoryEntry,
} from '@/lib/api/models/analytics';

export const sampleAppointment: Appointment = {
  id: 'appt-1',
  status: 'booked',
  providerId: 'prov-1',
  patientId: 'pat-1',
  timeSlot: '2026-07-06T15:00:00Z',
  version: 2,
};

export const sampleProvider: ProviderSummary = {
  id: 'prov-1',
  name: 'Dr. Amelia Ross',
  specialty: 'Family Medicine',
  clinicId: 'clinic-1',
  acceptingNew: true,
};

export const sampleProviderSchedule: ProviderSchedule = {
  providerId: 'prov-1',
  slots: ['2026-07-08T14:00:00Z', '2026-07-08T14:30:00Z', '2026-07-08T15:00:00Z'],
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

export const sampleNotification: Notification = {
  id: 'notif-1',
  kind: 'appointment.reminder',
  title: 'Visit tomorrow',
  body: 'Your telehealth visit with Dr. Ross is tomorrow at 10:00.',
  createdAt: '2026-07-05T09:00:00Z',
};

export const samplePrescription: Prescription = {
  id: 'rx-1',
  status: 'issued',
  patientId: 'pat-1',
  providerId: 'prov-1',
  medication: 'Amoxicillin 500mg',
  dosage: '1 capsule three times daily for 10 days',
  version: 3,
};

export const sampleIntakeForm: IntakeForm = {
  id: 'intake-1',
  status: 'pending',
  patientId: 'pat-1',
  appointmentId: 'appt-1',
  answers: [],
  version: 1,
};

export const sampleEncounter: Encounter = {
  id: 'enc-1',
  status: 'draft',
  patientId: 'pat-1',
  providerId: 'prov-1',
  appointmentId: 'appt-1',
  note: '',
  version: 1,
};

export const sampleLabOrder: LabOrder = {
  id: 'lab-1',
  status: 'ordered',
  encounterId: 'enc-1',
  patientId: 'pat-1',
  testCode: 'CBC',
  orderedAt: '2026-07-06T15:20:00Z',
  version: 1,
};

export const sampleInvoice: Invoice = {
  id: 'inv-1',
  status: 'issued',
  patientId: 'pat-1',
  total: { amountMinor: 12500, currency: 'USD' },
  issuedAt: '2026-07-01T00:00:00Z',
  version: 1,
};

export const samplePayment: Payment = {
  id: 'pay-1',
  status: 'captured',
  invoiceId: 'inv-1',
  amount: { amountMinor: 12500, currency: 'USD' },
  capturedAt: '2026-07-05T12:00:00Z',
  version: 1,
};

export const sampleDashboard: AnalyticsDashboard = {
  id: 'dash-1',
  name: 'Clinic Performance',
  refreshedAt: '2026-07-05T08:00:00Z',
  metrics: [
    { key: 'utilization', label: 'Utilization', value: 82, unit: '%' },
    { key: 'no_show_rate', label: 'No-show rate', value: 4.1, unit: '%' },
    { key: 'revenue', label: 'Revenue (MTD)', value: 248000, unit: 'USD' },
  ],
};

export const sampleClinic: ClinicDirectoryEntry = {
  id: 'clinic-1',
  name: 'Ancora Downtown',
  active: true,
  providerIds: ['prov-1', 'prov-2'],
};

export const handlers = [
  // --- Scheduling ------------------------------------------------------------
  http.get('*/api/scheduling/appointments', () =>
    HttpResponse.json([sampleAppointment]),
  ),
  http.get('*/api/scheduling/providers', () => HttpResponse.json([sampleProvider])),
  http.get('*/api/scheduling/providers/:id/schedule', () =>
    HttpResponse.json(sampleProviderSchedule),
  ),
  http.get('*/api/scheduling/appointments/:id', () =>
    HttpResponse.json(sampleAppointment),
  ),
  http.post('*/api/scheduling/appointments/hold', () =>
    HttpResponse.json(sampleAppointment),
  ),
  http.post('*/api/scheduling/appointments/:id/reschedule', () =>
    HttpResponse.json(sampleAppointment),
  ),
  http.post('*/api/scheduling/appointments/:id/cancel', () =>
    HttpResponse.json({ ...sampleAppointment, status: 'cancelled' }),
  ),
  http.post('*/api/scheduling/appointments/walk-in', () =>
    HttpResponse.json(sampleAppointment),
  ),

  // --- Clinical --------------------------------------------------------------
  http.get('*/api/clinical/encounters', () => HttpResponse.json([sampleEncounter])),
  http.get('*/api/clinical/encounters/:id/lab-orders', () =>
    HttpResponse.json([sampleLabOrder]),
  ),
  http.get('*/api/clinical/encounters/:id', () =>
    HttpResponse.json(sampleEncounter),
  ),
  http.post('*/api/clinical/encounters/:id/document', async ({ request }) => {
    const body = (await request.json()) as { note: string };
    return HttpResponse.json({ ...sampleEncounter, note: body.note });
  }),
  http.post('*/api/clinical/encounters/:id/sign', () =>
    HttpResponse.json({
      ...sampleEncounter,
      status: 'signed',
      signedAt: '2026-07-06T16:00:00Z',
    }),
  ),
  http.post('*/api/clinical/lab-orders', () => HttpResponse.json(sampleLabOrder)),

  // --- Engagement (REST + GraphQL) ------------------------------------------
  graphql.query('Threads', () =>
    HttpResponse.json({ data: { messageThreads: [sampleThread] } }),
  ),
  graphql.query('ThreadMessages', () =>
    HttpResponse.json({ data: { threadMessages: [sampleMessage] } }),
  ),
  http.post('*/api/engagement/threads/:id/messages', async ({ request }) => {
    const body = (await request.json()) as { body: string };
    return HttpResponse.json({ ...sampleMessage, id: 'msg-new', body: body.body });
  }),
  http.post('*/api/engagement/threads', () => HttpResponse.json(sampleThread)),
  http.get('*/api/engagement/notifications', () =>
    HttpResponse.json([sampleNotification]),
  ),
  http.get('*/api/engagement/prescriptions', () =>
    HttpResponse.json([samplePrescription]),
  ),
  http.post('*/api/engagement/prescriptions', async ({ request }) => {
    const body = (await request.json()) as { medication: string; dosage: string };
    return HttpResponse.json({
      ...samplePrescription,
      id: 'rx-new',
      status: 'drafted',
      medication: body.medication,
      dosage: body.dosage,
    });
  }),
  http.get('*/api/engagement/intake', () => HttpResponse.json([sampleIntakeForm])),
  http.post('*/api/engagement/intake', async ({ request }) => {
    const body = (await request.json()) as { answers: IntakeAnswer[] };
    return HttpResponse.json({
      ...sampleIntakeForm,
      status: 'submitted',
      submittedAt: '2026-07-05T10:00:00Z',
      answers: body.answers,
    });
  }),

  // --- Billing ---------------------------------------------------------------
  http.get('*/api/billing/invoices', () => HttpResponse.json([sampleInvoice])),
  http.get('*/api/billing/invoices/:id/payments', () =>
    HttpResponse.json([samplePayment]),
  ),
  http.get('*/api/billing/invoices/:id', () => HttpResponse.json(sampleInvoice)),
  http.post('*/api/billing/invoices/:id/payments', () =>
    HttpResponse.json(samplePayment),
  ),

  // --- Analytics -------------------------------------------------------------
  http.get('*/api/analytics/dashboards', () =>
    HttpResponse.json([sampleDashboard]),
  ),
  http.get('*/api/analytics/dashboards/:id', () =>
    HttpResponse.json(sampleDashboard),
  ),
  http.get('*/api/analytics/clinics', () => HttpResponse.json([sampleClinic])),
  http.post('*/api/analytics/clinics', async ({ request }) => {
    const body = (await request.json()) as { name: string };
    return HttpResponse.json({
      id: 'clinic-new',
      name: body.name,
      active: true,
      providerIds: [],
    });
  }),
];
