/**
 * Query-key factory.
 *
 * A single source of truth for TanStack Query cache keys, namespaced by bounded
 * context. Centralizing keys is what makes cache invalidation reliable: a
 * mutation invalidates `queryKeys.scheduling.appointments()` and every list/detail
 * query under that prefix is refetched, with no stringly-typed keys drifting
 * apart across files.
 */

import type { Id } from '../models/common';

export const queryKeys = {
  scheduling: {
    root: ['scheduling'] as const,
    appointments: () => [...queryKeys.scheduling.root, 'appointments'] as const,
    appointment: (id: Id) => [...queryKeys.scheduling.appointments(), id] as const,
    providers: () => [...queryKeys.scheduling.root, 'providers'] as const,
    providerSchedule: (providerId: Id) =>
      [...queryKeys.scheduling.root, 'provider-schedule', providerId] as const,
  },
  clinical: {
    root: ['clinical'] as const,
    encounters: () => [...queryKeys.clinical.root, 'encounters'] as const,
    encounter: (id: Id) => [...queryKeys.clinical.encounters(), id] as const,
    labOrders: (encounterId: Id) =>
      [...queryKeys.clinical.root, 'lab-orders', encounterId] as const,
  },
  engagement: {
    root: ['engagement'] as const,
    threads: () => [...queryKeys.engagement.root, 'threads'] as const,
    thread: (id: Id) => [...queryKeys.engagement.threads(), id] as const,
    messages: (threadId: Id) =>
      [...queryKeys.engagement.root, 'messages', threadId] as const,
    notifications: () => [...queryKeys.engagement.root, 'notifications'] as const,
    prescriptions: () => [...queryKeys.engagement.root, 'prescriptions'] as const,
    intake: () => [...queryKeys.engagement.root, 'intake'] as const,
  },
  billing: {
    root: ['billing'] as const,
    invoices: () => [...queryKeys.billing.root, 'invoices'] as const,
    invoice: (id: Id) => [...queryKeys.billing.invoices(), id] as const,
    payments: (invoiceId: Id) =>
      [...queryKeys.billing.root, 'payments', invoiceId] as const,
  },
  analytics: {
    root: ['analytics'] as const,
    dashboards: () => [...queryKeys.analytics.root, 'dashboards'] as const,
    dashboard: (id: Id) => [...queryKeys.analytics.dashboards(), id] as const,
    clinics: () => [...queryKeys.analytics.root, 'clinics'] as const,
  },
} as const;
