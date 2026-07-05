/**
 * Clinical bounded-context models.
 *
 * Aligned to `src/domain/clinicalrecords` — the Encounter and LabOrder
 * aggregates. Clinical fields are PHI; the client only ever transports what the
 * edge already authorized the caller to see.
 */

import type { Id, IsoDateTime } from './common';

/** Encounter lifecycle state. */
export type EncounterStatus = 'draft' | 'signed' | 'amended';

/** A clinical encounter / visit note. */
export interface Encounter {
  id: Id;
  status: EncounterStatus;
  patientId: Id;
  providerId: Id;
  appointmentId?: Id;
  /** Free-text clinical note body (PHI). */
  note: string;
  signedAt?: IsoDateTime;
  version: number;
}

/** Lab order lifecycle state. */
export type LabOrderStatus = 'ordered' | 'collected' | 'resulted' | 'cancelled';

/** A laboratory order attached to an encounter. */
export interface LabOrder {
  id: Id;
  status: LabOrderStatus;
  encounterId: Id;
  patientId: Id;
  /** LOINC/panel code for the test ordered. */
  testCode: string;
  orderedAt: IsoDateTime;
  version: number;
}

/** DocumentEncounterCmd: record/append a clinical note on an encounter. */
export interface DocumentEncounterRequest {
  encounterId: Id;
  note: string;
}

/** SignEncounterCmd: attest and lock an encounter note. */
export interface SignEncounterRequest {
  encounterId: Id;
}

/** PlaceLabOrderCmd: order a lab test for a patient on an encounter. */
export interface PlaceLabOrderRequest {
  encounterId: Id;
  patientId: Id;
  testCode: string;
}
