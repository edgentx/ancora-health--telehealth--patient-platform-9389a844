/**
 * Patient-engagement bounded-context models.
 *
 * Aligned to `src/domain/patientengagement` — MessageThread, IntakeForm and
 * Prescription aggregates. Secure messaging and notifications are the realtime
 * surfaces consumed over the WebSocket client as well as REST.
 */

import type { Id, IsoDateTime } from './common';

/** Message-thread lifecycle. */
export type MessageThreadStatus = 'new' | 'open';

/** A secure patient/care-team messaging thread. */
export interface MessageThread {
  id: Id;
  status: MessageThreadStatus;
  patientId: Id;
  careTeamMemberIds: Id[];
  subject: string;
  version: number;
}

/** A single message within a thread. */
export interface Message {
  id: Id;
  threadId: Id;
  authorId: Id;
  body: string;
  sentAt: IsoDateTime;
}

/** A realtime notification pushed over the `notifications` channel. */
export interface Notification {
  id: Id;
  kind: string;
  title: string;
  body: string;
  createdAt: IsoDateTime;
  readAt?: IsoDateTime;
}

/** Prescription lifecycle. */
export type PrescriptionStatus = 'drafted' | 'safety_checked' | 'issued';

/** An e-prescription. */
export interface Prescription {
  id: Id;
  status: PrescriptionStatus;
  patientId: Id;
  /** Provider who composed the order, when the read model exposes it. */
  providerId?: Id;
  medication: string;
  /** Dosage instruction drafted with the medication. */
  dosage?: string;
  version: number;
}

/** Intake-form lifecycle. */
export type IntakeFormStatus = 'pending' | 'submitted';

/** A single answered field on a pre-visit intake form. */
export interface IntakeAnswer {
  /** Stable field key (e.g. `chief_complaint`, `allergies`). */
  key: string;
  /** Human-facing prompt shown next to the answer. */
  label: string;
  /** The patient's answer. */
  value: string;
}

/** A patient-completed pre-visit intake form. */
export interface IntakeForm {
  id: Id;
  status: IntakeFormStatus;
  patientId: Id;
  appointmentId?: Id;
  answers: IntakeAnswer[];
  submittedAt?: IsoDateTime;
  version: number;
}

/** StartMessageThreadCmd: open a secure thread with the care team. */
export interface StartMessageThreadRequest {
  patientId: Id;
  careTeamMemberIds: Id[];
  subject: string;
}

/** PostMessageCmd: append a message to an open thread. */
export interface PostMessageRequest {
  threadId: Id;
  body: string;
}

/**
 * ComposePrescriptionCmd: draft a new prescription for a patient. Mirrors the
 * backend command's mandatory fields — patient, provider, medication, dosage.
 */
export interface ComposePrescriptionRequest {
  patientId: Id;
  providerId: Id;
  medication: string;
  dosage: string;
}

/** SubmitIntakeCmd: submit a patient's completed pre-visit intake answers. */
export interface SubmitIntakeRequest {
  patientId: Id;
  appointmentId?: Id;
  answers: IntakeAnswer[];
}
