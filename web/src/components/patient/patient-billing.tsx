'use client';

import { useState } from 'react';

import { useCapturePayment, useInvoices, type Invoice } from '@/lib/api';
import { formatMoney } from '@/lib/format';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/**
 * Patient billing: outstanding invoices and a one-click payment that captures
 * the full balance. "Initiating payment" here is the CapturePaymentCmd against
 * the invoice; a captured payment invalidates the invoice so its status flips.
 */
export function PatientBilling() {
  const invoicesQuery = useInvoices();

  return (
    <section>
      <h1 className="page-heading">Billing &amp; invoices</h1>
      <p className="page-subheading">Review balances and pay open invoices.</p>

      <QueryState
        query={invoicesQuery}
        isEmpty={(invoices) => invoices.length === 0}
        loadingLabel="Loading invoices…"
        emptyLabel="You have no invoices — nothing due."
      >
        {(invoices) => (
          <ul className="list">
            {invoices.map((invoice) => (
              <InvoiceRow key={invoice.id} invoice={invoice} />
            ))}
          </ul>
        )}
      </QueryState>
    </section>
  );
}

function InvoiceRow({ invoice }: { invoice: Invoice }) {
  const capture = useCapturePayment();
  const [paid, setPaid] = useState(false);
  const payable = invoice.status === 'issued' && !paid;

  function pay() {
    capture.mutate(
      { invoiceId: invoice.id, amount: invoice.total },
      { onSuccess: () => setPaid(true) },
    );
  }

  return (
    <li className="list__row">
      <span>
        <span className="list__primary">{formatMoney(invoice.total)}</span>
        <span className="list__meta"> · invoice {invoice.id}</span>
      </span>
      <span style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
        <StatusPill status={paid ? 'paid' : invoice.status} />
        {capture.isError ? (
          <span className="list__meta" role="alert" style={{ color: '#b91c1c' }}>
            Payment failed
          </span>
        ) : null}
        {payable ? (
          <button
            type="button"
            className="btn btn--primary"
            disabled={capture.isPending}
            onClick={pay}
          >
            {capture.isPending ? 'Processing…' : 'Pay now'}
          </button>
        ) : null}
      </span>
    </li>
  );
}
