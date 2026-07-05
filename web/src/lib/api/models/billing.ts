/**
 * Billing & insurance bounded-context models.
 *
 * Aligned to `src/domain/billingandinsurance` — Invoice, Payment and
 * InsurancePolicy aggregates. Amounts use the integer-minor-unit {@link Money}
 * type to avoid floating-point drift on currency.
 */

import type { Id, IsoDateTime, Money } from './common';

/** Invoice lifecycle. */
export type InvoiceStatus = 'draft' | 'issued' | 'paid' | 'void';

/** A patient/payer invoice. */
export interface Invoice {
  id: Id;
  status: InvoiceStatus;
  patientId: Id;
  total: Money;
  issuedAt?: IsoDateTime;
  version: number;
}

/** Payment lifecycle. */
export type PaymentStatus = 'pending' | 'captured' | 'refunded' | 'failed';

/** A payment applied against an invoice. */
export interface Payment {
  id: Id;
  status: PaymentStatus;
  invoiceId: Id;
  amount: Money;
  capturedAt?: IsoDateTime;
  version: number;
}

/** An insurance policy on file for a patient. */
export interface InsurancePolicy {
  id: Id;
  patientId: Id;
  payerId: Id;
  memberNumber: string;
  active: boolean;
  version: number;
}

/** IssueInvoiceCmd: finalize a draft invoice for collection. */
export interface IssueInvoiceRequest {
  invoiceId: Id;
}

/** CapturePaymentCmd: capture a payment against an invoice. */
export interface CapturePaymentRequest {
  invoiceId: Id;
  amount: Money;
}

/** VerifyEligibilityRequest: check a policy's active coverage. */
export interface VerifyEligibilityRequest {
  policyId: Id;
}
