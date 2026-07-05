'use client';

import { useState } from 'react';

import {
  ApiError,
  useAppointments,
  useHoldSlot,
  useProviders,
  useProviderSchedule,
  useRescheduleAppointment,
} from '@/lib/api';
import { formatDateTime } from '@/lib/format';

import { QueryState } from '../query-state';
import { StatusPill } from '../status-pill';

/**
 * Front-desk scheduler. Pick a provider to see their open slots, then either
 * book a new appointment for a patient (HoldSlotCmd) or reschedule an existing
 * one (RescheduleAppointmentCmd) into a chosen slot. The backend rejects a
 * double-book with 409; both flows share {@link conflictMessage} so a taken slot
 * always surfaces the same clear, recoverable message.
 */
export function SchedulerCalendar() {
  const providersQuery = useProviders();
  const appointmentsQuery = useAppointments();
  const [providerId, setProviderId] = useState('');
  const [patientId, setPatientId] = useState('');
  // When set, the next slot click reschedules this appointment instead of
  // booking a new one.
  const [rescheduleId, setRescheduleId] = useState<string | null>(null);

  const hold = useHoldSlot();
  const reschedule = useRescheduleAppointment();

  const activeProvider =
    providersQuery.data?.find((p) => p.id === providerId) ?? providersQuery.data?.[0];
  const effectiveProviderId = providerId || activeProvider?.id || '';

  function onSlot(slot: string) {
    if (rescheduleId) {
      reschedule.mutate(
        { appointmentId: rescheduleId, newTimeSlot: slot },
        { onSuccess: () => setRescheduleId(null) },
      );
    } else {
      hold.mutate({ providerId: effectiveProviderId, patientId: patientId.trim() || 'walk-in', timeSlot: slot });
    }
  }

  const conflict = conflictMessage(hold.error) ?? conflictMessage(reschedule.error);
  const booked = hold.isSuccess || reschedule.isSuccess;

  return (
    <section>
      <h1 className="page-heading">Schedule management</h1>
      <p className="page-subheading">Book and reschedule appointments across providers.</p>

      <div className="section-head">
        <div style={{ display: 'flex', gap: 'var(--space-3)', flexWrap: 'wrap' }}>
          <div className="field" style={{ margin: 0 }}>
            <label className="field__label" htmlFor="sched-provider">
              Provider
            </label>
            <select
              id="sched-provider"
              className="select"
              value={effectiveProviderId}
              onChange={(e) => setProviderId(e.target.value)}
            >
              {(providersQuery.data ?? []).map((provider) => (
                <option key={provider.id} value={provider.id}>
                  {provider.name}
                </option>
              ))}
            </select>
          </div>
          <div className="field" style={{ margin: 0 }}>
            <label className="field__label" htmlFor="sched-patient">
              Patient ID (for new bookings)
            </label>
            <input
              id="sched-patient"
              className="input"
              value={patientId}
              onChange={(e) => setPatientId(e.target.value)}
            />
          </div>
        </div>
      </div>

      {conflict ? (
        <p className="banner banner--conflict" role="alert">
          {conflict}
        </p>
      ) : null}
      {booked && !conflict ? (
        <p className="banner banner--success" role="status">
          {reschedule.isSuccess
            ? 'Appointment updated.'
            : 'Slot held — the appointment is booked.'}
        </p>
      ) : null}
      {rescheduleId ? (
        <p className="banner banner--info" role="status">
          Rescheduling appointment {rescheduleId} — choose a new time below.{' '}
          <button
            type="button"
            className="btn btn--ghost"
            onClick={() => setRescheduleId(null)}
          >
            Cancel
          </button>
        </p>
      ) : null}

      <div className="split">
        <div className="stack">
          <h2 style={{ margin: 0, fontSize: '1.125rem' }}>Appointments</h2>
          <QueryState
            query={appointmentsQuery}
            isEmpty={(appointments) => appointments.length === 0}
            loadingLabel="Loading appointments…"
            emptyLabel="No appointments booked."
          >
            {(appointments) => (
              <ul className="list">
                {appointments.map((appointment) => (
                  <li key={appointment.id} className="list__row">
                    <span>
                      <span className="list__primary">{formatDateTime(appointment.timeSlot)}</span>
                      <span className="list__meta"> · patient {appointment.patientId}</span>
                    </span>
                    <span style={{ display: 'flex', gap: 'var(--space-2)', alignItems: 'center' }}>
                      <StatusPill status={appointment.status} />
                      {appointment.status !== 'cancelled' ? (
                        <button
                          type="button"
                          className="btn btn--ghost"
                          onClick={() => setRescheduleId(appointment.id)}
                        >
                          Reschedule
                        </button>
                      ) : null}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </QueryState>
        </div>

        <div className="stack">
          <h2 style={{ margin: 0, fontSize: '1.125rem' }}>Open slots</h2>
          {effectiveProviderId ? (
            <SlotPicker
              providerId={effectiveProviderId}
              disabled={hold.isPending || reschedule.isPending}
              onSlot={onSlot}
            />
          ) : (
            <div className="qs qs--loading" role="status">
              <span className="qs__spinner" aria-hidden />
              Loading slots…
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

/**
 * Loads and renders a provider's open slots. Isolated as a child so it mounts
 * with a concrete provider id — a fresh, enabled query — rather than starting
 * disabled at the parent and relying on an empty-string → id transition.
 */
function SlotPicker({
  providerId,
  disabled,
  onSlot,
}: {
  providerId: string;
  disabled: boolean;
  onSlot: (slot: string) => void;
}) {
  const scheduleQuery = useProviderSchedule(providerId);
  return (
    <QueryState
      query={scheduleQuery}
      isEmpty={(schedule) => schedule.slots.length === 0}
      loadingLabel="Loading slots…"
      emptyLabel="No open slots for this provider."
    >
      {(schedule) => (
        <div className="slot-grid">
          {schedule.slots.map((slot) => (
            <button
              key={slot}
              type="button"
              className="slot"
              disabled={disabled}
              onClick={() => onSlot(slot)}
            >
              {formatDateTime(slot)}
            </button>
          ))}
        </div>
      )}
    </QueryState>
  );
}

/**
 * Turn a booking/reschedule error into a scheduler-facing message. A 409 is the
 * double-book case and gets an explicit "already taken" message; other errors
 * fall back to a generic retryable message.
 */
export function conflictMessage(error: unknown): string | null {
  if (!(error instanceof ApiError)) return null;
  if (error.status === 409) {
    return 'That slot was just taken by another booking. Please choose a different time.';
  }
  return 'We couldn’t complete that change. Please try again.';
}
