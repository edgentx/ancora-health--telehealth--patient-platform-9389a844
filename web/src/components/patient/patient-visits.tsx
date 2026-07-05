'use client';

import { useAppointments } from '@/lib/api';
import { formatDateTime } from '@/lib/format';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/** Patient's upcoming and past visits, read from the scheduling read model. */
export function PatientVisits() {
  const appointmentsQuery = useAppointments();

  return (
    <section>
      <h1 className="page-heading">Your visits</h1>
      <p className="page-subheading">Scheduled and recent telehealth appointments.</p>

      <QueryState
        query={appointmentsQuery}
        isEmpty={(appointments) => appointments.length === 0}
        loadingLabel="Loading your visits…"
        emptyLabel="You have no visits scheduled. Find a provider to book one."
      >
        {(appointments) => (
          <table className="table">
            <thead>
              <tr>
                <th>When</th>
                <th>Provider</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {appointments.map((appointment) => (
                <tr key={appointment.id}>
                  <td>{formatDateTime(appointment.timeSlot)}</td>
                  <td>{appointment.providerId}</td>
                  <td>
                    <StatusPill status={appointment.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </QueryState>
    </section>
  );
}
