'use client';

import { useState } from 'react';

import {
  ApiError,
  useHoldSlot,
  useProviders,
  useProviderSchedule,
  type ProviderSummary,
} from '@/lib/api';
import { formatDateTime } from '@/lib/format';
import { useUiStore } from '@/store/ui-store';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/**
 * Patient provider-discovery + booking. Pick a provider from the directory, load
 * their open slots, and hold one — a hold is the first step of booking in the
 * scheduling model. Success and failure both surface inline so the patient knows
 * whether the slot is theirs.
 */
export function ProviderDiscovery() {
  const providersQuery = useProviders();
  const [selected, setSelected] = useState<ProviderSummary | null>(null);

  return (
    <section>
      <h1 className="page-heading">Find a provider</h1>
      <p className="page-subheading">
        Choose a clinician and book an open visit time.
      </p>

      <div className="split">
        <QueryState
          query={providersQuery}
          isEmpty={(providers) => providers.length === 0}
          loadingLabel="Loading providers…"
          emptyLabel="No providers are accepting bookings right now."
        >
          {(providers) => {
            const active = selected ?? providers[0] ?? null;
            return (
              <ul className="list" aria-label="Providers">
                {providers.map((provider) => (
                  <li key={provider.id}>
                    <button
                      type="button"
                      className={`list__row${active?.id === provider.id ? ' is-active' : ''}`}
                      style={{ width: '100%', cursor: 'pointer', textAlign: 'left' }}
                      aria-pressed={active?.id === provider.id}
                      onClick={() => setSelected(provider)}
                    >
                      <span>
                        <span className="list__primary">{provider.name}</span>
                        <span className="list__meta"> · {provider.specialty}</span>
                      </span>
                      {provider.acceptingNew ? (
                        <StatusPill status="active">Accepting</StatusPill>
                      ) : (
                        <StatusPill status="inactive">Full</StatusPill>
                      )}
                    </button>
                  </li>
                ))}
              </ul>
            );
          }}
        </QueryState>

        {providersQuery.data && (selected ?? providersQuery.data[0]) ? (
          <BookingPane provider={selected ?? providersQuery.data[0]!} />
        ) : (
          <div className="qs qs--empty">Select a provider to see availability.</div>
        )}
      </div>
    </section>
  );
}

function BookingPane({ provider }: { provider: ProviderSummary }) {
  const scheduleQuery = useProviderSchedule(provider.id);
  const hold = useHoldSlot();
  const patientId = useUiStore((s) => s.user) ?? 'self';
  const [chosen, setChosen] = useState<string | null>(null);

  function book(slot: string) {
    setChosen(slot);
    hold.mutate({ providerId: provider.id, patientId, timeSlot: slot });
  }

  return (
    <div className="card">
      <div className="section-head" style={{ marginBottom: 'var(--space-3)' }}>
        <h2 style={{ margin: 0, fontSize: '1.125rem' }}>{provider.name}</h2>
        <span className="list__meta">{provider.specialty}</span>
      </div>

      {hold.isSuccess ? (
        <p className="banner banner--success" role="status">
          Booked — your visit on {chosen ? formatDateTime(chosen) : 'the selected slot'} is
          held. You’ll find it under Appointments.
        </p>
      ) : null}
      {hold.isError ? (
        <p className="banner banner--conflict" role="alert">
          {hold.error instanceof ApiError && hold.error.status === 409
            ? 'That time was just taken. Please choose another slot.'
            : 'We couldn’t hold that slot. Please try again.'}
        </p>
      ) : null}

      <QueryState
        query={scheduleQuery}
        isEmpty={(schedule) => schedule.slots.length === 0}
        loadingLabel="Loading availability…"
        emptyLabel="No open times — check back soon."
      >
        {(schedule) => (
          <div className="slot-grid" style={{ marginTop: 'var(--space-3)' }}>
            {schedule.slots.map((slot) => (
              <button
                key={slot}
                type="button"
                className={`slot${chosen === slot ? ' is-selected' : ''}`}
                disabled={hold.isPending}
                onClick={() => book(slot)}
              >
                {formatDateTime(slot)}
              </button>
            ))}
          </div>
        )}
      </QueryState>
    </div>
  );
}
