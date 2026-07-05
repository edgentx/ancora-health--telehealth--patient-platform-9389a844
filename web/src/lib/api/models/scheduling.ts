/**
 * Scheduling bounded-context models.
 *
 * Aligned to `src/domain/scheduling` — the Appointment and ProviderSchedule
 * aggregates and their commands (HoldSlotCmd, RescheduleAppointmentCmd,
 * CancelAppointmentCmd, RegisterWalkInCmd). Request DTOs carry exactly the
 * fields those commands require.
 */

import type { Id, IsoDateTime } from './common';

/** Appointment lifecycle, mirroring `AppointmentStatus` on the backend. */
export type AppointmentStatus = 'open' | 'held' | 'booked' | 'cancelled';

/** A scheduling appointment as returned by the REST/GraphQL read models. */
export interface Appointment {
  id: Id;
  status: AppointmentStatus;
  providerId: Id;
  patientId: Id;
  timeSlot: IsoDateTime;
  clinicId?: Id;
  version: number;
}

/** A provider's published availability window. */
export interface ProviderSchedule {
  providerId: Id;
  slots: IsoDateTime[];
}

/**
 * A discoverable provider directory entry, used by the patient booking flow to
 * choose who to book with before loading that provider's open slots.
 */
export interface ProviderSummary {
  id: Id;
  name: string;
  specialty: string;
  clinicId?: Id;
  /** Whether the provider is currently accepting new bookings. */
  acceptingNew: boolean;
}

/** HoldSlotCmd: reserve a provider slot for a patient. */
export interface HoldSlotRequest {
  providerId: Id;
  timeSlot: IsoDateTime;
  patientId: Id;
}

/** RescheduleAppointmentCmd: move an appointment to a new slot within policy. */
export interface RescheduleAppointmentRequest {
  appointmentId: Id;
  newTimeSlot: IsoDateTime;
}

/** CancelAppointmentCmd: cancel an appointment within the policy window. */
export interface CancelAppointmentRequest {
  appointmentId: Id;
  reason?: string;
}

/** RegisterWalkInCmd: register an unscheduled patient at the front desk. */
export interface RegisterWalkInRequest {
  patientId: Id;
  clinicId: Id;
  providerId: Id;
}
