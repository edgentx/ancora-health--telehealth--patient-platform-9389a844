/**
 * Billing & insurance hooks (invoices, payments, eligibility).
 *
 * Issuing an invoice or capturing a payment invalidates the relevant keys so
 * balances and payment lists stay consistent with the backend.
 */
'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '../client';
import type {
  CapturePaymentRequest,
  InsurancePolicy,
  Invoice,
  IssueInvoiceRequest,
  Payment,
  VerifyEligibilityRequest,
} from '../models/billing';
import { queryKeys } from './keys';

export function useInvoices() {
  return useQuery({
    queryKey: queryKeys.billing.invoices(),
    queryFn: () => apiClient.rest.get<Invoice[]>('/api/billing/invoices'),
  });
}

export function useInvoice(id: string) {
  return useQuery({
    queryKey: queryKeys.billing.invoice(id),
    queryFn: () => apiClient.rest.get<Invoice>(`/api/billing/invoices/${id}`),
    enabled: id.length > 0,
  });
}

export function usePayments(invoiceId: string) {
  return useQuery({
    queryKey: queryKeys.billing.payments(invoiceId),
    queryFn: () =>
      apiClient.rest.get<Payment[]>(`/api/billing/invoices/${invoiceId}/payments`),
    enabled: invoiceId.length > 0,
  });
}

export function useIssueInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: IssueInvoiceRequest) =>
      apiClient.rest.post<Invoice>(
        `/api/billing/invoices/${input.invoiceId}/issue`,
        input,
      ),
    onSuccess: (invoice) => {
      void qc.invalidateQueries({ queryKey: queryKeys.billing.invoices() });
      void qc.invalidateQueries({ queryKey: queryKeys.billing.invoice(invoice.id) });
    },
  });
}

export function useCapturePayment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CapturePaymentRequest) =>
      apiClient.rest.post<Payment>(
        `/api/billing/invoices/${input.invoiceId}/payments`,
        input,
      ),
    onSuccess: (payment) => {
      void qc.invalidateQueries({
        queryKey: queryKeys.billing.payments(payment.invoiceId),
      });
      void qc.invalidateQueries({
        queryKey: queryKeys.billing.invoice(payment.invoiceId),
      });
    },
  });
}

export function useVerifyEligibility() {
  return useMutation({
    mutationFn: (input: VerifyEligibilityRequest) =>
      apiClient.rest.post<InsurancePolicy>(
        `/api/billing/policies/${input.policyId}/eligibility`,
        input,
      ),
  });
}
