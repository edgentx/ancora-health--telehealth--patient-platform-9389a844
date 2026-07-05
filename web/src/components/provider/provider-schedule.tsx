'use client';

import Link from 'next/link';

import { useAppointments } from '@/lib/api';
import { formatDateTime } from '@/lib/format';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/**
 * The provider's day: appointments the edge has already scoped to this
 * clinician. Each booked visit offers a "Start visit" link into the WebRTC
 * room, keyed by the appointment id so the signaling session is per-visit.
 */
export function ProviderScheduleView() {
  const appointmentsQuery = useAppointments();

  return (
    <section>
      <h1 className="page-heading">Today’s schedule</h1>
      <p className="page-subheading">Your booked visits and their status.</p>

      <QueryState
        query={appointmentsQuery}
        isEmpty={(appointments) => appointments.length === 0}
        loadingLabel="Loading your schedule…"
        emptyLabel="No visits on the schedule."
      >
        {(appointments) => (
          <table className="table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Patient</th>
                <th>Status</th>
                <th aria-label="Actions" />
              </tr>
            </thead>
            <tbody>
              {appointments.map((appointment) => (
                <tr key={appointment.id}>
                  <td>{formatDateTime(appointment.timeSlot)}</td>
                  <td>{appointment.patientId}</td>
                  <td>
                    <StatusPill status={appointment.status} />
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    {appointment.status === 'booked' ? (
                      <Link
                        className="btn btn--primary"
                        href={`/provider/visit/${appointment.id}`}
                      >
                        Start visit
                      </Link>
                    ) : null}
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
