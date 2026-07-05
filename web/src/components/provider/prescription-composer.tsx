'use client';

import { useState } from 'react';

import { useComposePrescription, usePrescriptions } from '@/lib/api';
import { useUiStore } from '@/store/ui-store';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/**
 * E-prescribing composer. The provider drafts a prescription
 * (ComposePrescriptionCmd needs patient, provider, medication, dosage); a
 * successful compose invalidates the prescriptions list so the new draft appears
 * below. Safety checks and transmission are downstream acts on the aggregate.
 */
export function PrescriptionComposer() {
  const compose = useComposePrescription();
  const prescriptionsQuery = usePrescriptions();
  const providerId = useUiStore((s) => s.user) ?? 'self';

  const [patientId, setPatientId] = useState('');
  const [medication, setMedication] = useState('');
  const [dosage, setDosage] = useState('');

  const canSubmit =
    patientId.trim().length > 0 &&
    medication.trim().length > 0 &&
    dosage.trim().length > 0 &&
    !compose.isPending;

  function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    if (!canSubmit) return;
    compose.mutate(
      {
        patientId: patientId.trim(),
        providerId,
        medication: medication.trim(),
        dosage: dosage.trim(),
      },
      {
        onSuccess: () => {
          setMedication('');
          setDosage('');
        },
      },
    );
  }

  return (
    <section>
      <h1 className="page-heading">E-prescribing</h1>
      <p className="page-subheading">Compose a new prescription for a patient.</p>

      <div className="split">
        <form className="card" onSubmit={onSubmit} aria-label="Compose prescription">
          <div className="field">
            <label className="field__label" htmlFor="rx-patient">
              Patient ID
            </label>
            <input
              id="rx-patient"
              className="input"
              value={patientId}
              onChange={(e) => setPatientId(e.target.value)}
            />
          </div>
          <div className="field">
            <label className="field__label" htmlFor="rx-medication">
              Medication
            </label>
            <input
              id="rx-medication"
              className="input"
              value={medication}
              onChange={(e) => setMedication(e.target.value)}
            />
          </div>
          <div className="field">
            <label className="field__label" htmlFor="rx-dosage">
              Dosage &amp; directions
            </label>
            <input
              id="rx-dosage"
              className="input"
              value={dosage}
              onChange={(e) => setDosage(e.target.value)}
            />
          </div>

          {compose.isSuccess ? (
            <p className="banner banner--success" role="status">
              Prescription drafted. Run a safety check before transmitting.
            </p>
          ) : null}
          {compose.isError ? (
            <p className="banner banner--conflict" role="alert">
              We couldn’t draft that prescription. Please check the details and retry.
            </p>
          ) : null}

          <div className="form-actions">
            <button type="submit" className="btn btn--primary" disabled={!canSubmit}>
              {compose.isPending ? 'Drafting…' : 'Compose prescription'}
            </button>
          </div>
        </form>

        <div className="card">
          <h2 style={{ marginTop: 0, fontSize: '1.125rem' }}>Recent prescriptions</h2>
          <QueryState
            query={prescriptionsQuery}
            isEmpty={(prescriptions) => prescriptions.length === 0}
            loadingLabel="Loading prescriptions…"
            emptyLabel="No prescriptions yet."
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
        </div>
      </div>
    </section>
  );
}
