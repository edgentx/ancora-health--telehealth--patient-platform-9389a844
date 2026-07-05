'use client';

import { usePrescriptions } from '@/lib/api';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/** Read-only list of the patient's prescriptions and their fulfilment status. */
export function PatientPrescriptions() {
  const prescriptionsQuery = usePrescriptions();

  return (
    <section>
      <h1 className="page-heading">Prescriptions</h1>
      <p className="page-subheading">Medications your providers have prescribed.</p>

      <QueryState
        query={prescriptionsQuery}
        isEmpty={(prescriptions) => prescriptions.length === 0}
        loadingLabel="Loading prescriptions…"
        emptyLabel="You have no prescriptions on file."
      >
        {(prescriptions) => (
          <ul className="list">
            {prescriptions.map((rx) => (
              <li key={rx.id} className="list__row">
                <span>
                  <span className="list__primary">{rx.medication}</span>
                  {rx.dosage ? <span className="list__meta"> · {rx.dosage}</span> : null}
                </span>
                <StatusPill status={rx.status} />
              </li>
            ))}
          </ul>
        )}
      </QueryState>
    </section>
  );
}
