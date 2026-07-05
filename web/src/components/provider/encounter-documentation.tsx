'use client';

import { useEffect, useState } from 'react';

import {
  useDocumentEncounter,
  useEncounters,
  useLabOrders,
  usePlaceLabOrder,
  useSignEncounter,
  type Encounter,
} from '@/lib/api';
import { formatDateTime } from '@/lib/format';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/**
 * Clinical documentation. Pick an encounter, write/append its note
 * (DocumentEncounterCmd), sign it (SignEncounterCmd — which locks the note), and
 * place lab orders against it. Signing is disabled once the encounter is signed,
 * reflecting the aggregate's immutability invariant.
 */
export function EncounterDocumentation() {
  const encountersQuery = useEncounters();
  const [activeId, setActiveId] = useState<string | null>(null);

  return (
    <section>
      <h1 className="page-heading">Clinical notes</h1>
      <p className="page-subheading">Document encounters and order labs.</p>

      <QueryState
        query={encountersQuery}
        isEmpty={(encounters) => encounters.length === 0}
        loadingLabel="Loading encounters…"
        emptyLabel="No encounters to document."
      >
        {(encounters) => {
          const active =
            encounters.find((e) => e.id === activeId) ?? encounters[0] ?? null;
          return (
            <div className="split">
              <ul className="list" aria-label="Encounters">
                {encounters.map((encounter) => (
                  <li key={encounter.id}>
                    <button
                      type="button"
                      className={`list__row${active?.id === encounter.id ? ' is-active' : ''}`}
                      style={{ width: '100%', cursor: 'pointer', textAlign: 'left' }}
                      aria-pressed={active?.id === encounter.id}
                      onClick={() => setActiveId(encounter.id)}
                    >
                      <span>
                        <span className="list__primary">Patient {encounter.patientId}</span>
                      </span>
                      <StatusPill status={encounter.status} />
                    </button>
                  </li>
                ))}
              </ul>
              {active ? <EncounterEditor encounter={active} /> : null}
            </div>
          );
        }}
      </QueryState>
    </section>
  );
}

function EncounterEditor({ encounter }: { encounter: Encounter }) {
  const document = useDocumentEncounter();
  const sign = useSignEncounter();
  const [note, setNote] = useState(encounter.note);
  const signed = (sign.data ?? encounter).status === 'signed';

  // Reset the editor when switching to a different encounter.
  useEffect(() => setNote(encounter.note), [encounter.id, encounter.note]);

  return (
    <div className="card stack">
      <div className="section-head" style={{ marginBottom: 0 }}>
        <h2 style={{ margin: 0, fontSize: '1.125rem' }}>Encounter note</h2>
        <StatusPill status={signed ? 'signed' : encounter.status} />
      </div>

      <div className="field">
        <label className="field__label" htmlFor="encounter-note">
          Clinical note
        </label>
        <textarea
          id="encounter-note"
          className="textarea"
          value={note}
          disabled={signed}
          onChange={(e) => setNote(e.target.value)}
        />
      </div>

      {sign.isSuccess ? (
        <p className="banner banner--success" role="status">
          Signed {formatDateTime(sign.data?.signedAt)} — this note is now locked.
        </p>
      ) : null}

      <div className="form-actions">
        <button
          type="button"
          className="btn btn--primary"
          disabled={signed || document.isPending || note.trim().length === 0}
          onClick={() => document.mutate({ encounterId: encounter.id, note })}
        >
          {document.isPending ? 'Saving…' : 'Save note'}
        </button>
        <button
          type="button"
          className="btn btn--ghost"
          disabled={signed || sign.isPending}
          onClick={() => sign.mutate({ encounterId: encounter.id })}
        >
          {sign.isPending ? 'Signing…' : 'Sign encounter'}
        </button>
      </div>

      <LabOrders encounter={encounter} />
    </div>
  );
}

function LabOrders({ encounter }: { encounter: Encounter }) {
  const labOrdersQuery = useLabOrders(encounter.id);
  const place = usePlaceLabOrder();
  const [testCode, setTestCode] = useState('');

  function order() {
    const code = testCode.trim();
    if (!code) return;
    place.mutate(
      { encounterId: encounter.id, patientId: encounter.patientId, testCode: code },
      { onSuccess: () => setTestCode('') },
    );
  }

  return (
    <div className="stack">
      <h3 style={{ margin: 0, fontSize: '1rem' }}>Lab orders</h3>

      <QueryState
        query={labOrdersQuery}
        isEmpty={(orders) => orders.length === 0}
        loadingLabel="Loading lab orders…"
        emptyLabel="No labs ordered for this encounter."
      >
        {(orders) => (
          <ul className="list">
            {orders.map((order) => (
              <li key={order.id} className="list__row">
                <span className="list__primary">{order.testCode}</span>
                <StatusPill status={order.status} />
              </li>
            ))}
          </ul>
        )}
      </QueryState>

      <div className="composer">
        <input
          className="input"
          aria-label="Test code"
          placeholder="Test code (e.g. CBC)"
          value={testCode}
          onChange={(e) => setTestCode(e.target.value)}
        />
        <button
          type="button"
          className="btn btn--primary"
          disabled={place.isPending || testCode.trim().length === 0}
          onClick={order}
        >
          {place.isPending ? 'Ordering…' : 'Order lab'}
        </button>
      </div>
    </div>
  );
}
