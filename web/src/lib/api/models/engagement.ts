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
  medication: string;
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
