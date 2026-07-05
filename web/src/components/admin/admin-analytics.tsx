'use client';

import { useState } from 'react';

import { useClinics, useDashboards, useRegisterClinic } from '@/lib/api';
import { formatDateTime, formatMetric } from '@/lib/format';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/**
 * Clinic admin dashboard: utilization / no-show / revenue from the analytics
 * dashboards, plus the clinic directory with inline registration. Metrics render
 * through {@link formatMetric} so a `%` or `USD` unit hint formats consistently.
 */
export function AdminAnalytics() {
  const dashboardsQuery = useDashboards();

  return (
    <section>
      <h1 className="page-heading">Analytics</h1>
      <p className="page-subheading">Utilization, no-show rate, and revenue across the clinic.</p>

      <QueryState
        query={dashboardsQuery}
        isEmpty={(dashboards) => dashboards.length === 0}
        loadingLabel="Loading analytics…"
        emptyLabel="No analytics have been published yet."
      >
        {(dashboards) => {
          const dashboard = dashboards[0]!;
          return (
            <>
              <div className="section-head">
                <h2 style={{ margin: 0, fontSize: '1.125rem' }}>{dashboard.name}</h2>
                <span className="list__meta">
                  Refreshed {formatDateTime(dashboard.refreshedAt)}
                </span>
              </div>
              <div className="card-grid">
                {dashboard.metrics.map((metric) => (
                  <article className="card" key={metric.key}>
                    <p className="card__label">{metric.label}</p>
                    <p className="card__value">{formatMetric(metric.value, metric.unit)}</p>
                  </article>
                ))}
              </div>
            </>
          );
        }}
      </QueryState>

      <div style={{ marginTop: 'var(--space-8)' }}>
        <ClinicDirectory />
      </div>
    </section>
  );
}

function ClinicDirectory() {
  const clinicsQuery = useClinics();
  const register = useRegisterClinic();
  const [name, setName] = useState('');

  const canRegister = name.trim().length > 0 && !register.isPending;

  function onRegister(event: React.FormEvent) {
    event.preventDefault();
    if (!canRegister) return;
    register.mutate({ name: name.trim() }, { onSuccess: () => setName('') });
  }

  return (
    <div className="stack">
      <div className="section-head" style={{ marginBottom: 0 }}>
        <h2 style={{ margin: 0, fontSize: '1.125rem' }}>Clinic directory</h2>
      </div>

      <form className="composer" onSubmit={onRegister} aria-label="Register clinic">
        <input
          className="input"
          aria-label="Clinic name"
          placeholder="New clinic name"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button type="submit" className="btn btn--primary" disabled={!canRegister}>
          {register.isPending ? 'Adding…' : 'Register clinic'}
        </button>
      </form>

      <QueryState
        query={clinicsQuery}
        isEmpty={(clinics) => clinics.length === 0}
        loadingLabel="Loading clinics…"
        emptyLabel="No clinics in the directory yet."
      >
        {(clinics) => (
          <table className="table">
            <thead>
              <tr>
                <th>Clinic</th>
                <th>Providers</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {clinics.map((clinic) => (
                <tr key={clinic.id}>
                  <td>{clinic.name}</td>
                  <td>{clinic.providerIds.length}</td>
                  <td>
                    <StatusPill status={clinic.active ? 'active' : 'inactive'} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </QueryState>
    </div>
  );
}
