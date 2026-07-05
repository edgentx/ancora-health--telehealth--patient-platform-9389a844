/**
 * Scheduling query/mutation hooks.
 *
 * Reads use `useQuery`; each mutation (hold, reschedule, cancel, walk-in)
 * invalidates the appointments key on success so any list/detail view refetches
 * the authoritative state. Hooks are the only thing view stories import — they
 * never touch `apiClient` directly.
 */
'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '../client';
import type {
  Appointment,
  CancelAppointmentRequest,
  HoldSlotRequest,
  ProviderSchedule,
  RegisterWalkInRequest,
  RescheduleAppointmentRequest,
} from '../models/scheduling';
import { queryKeys } from './keys';

export function useAppointments() {
  return useQuery({
    queryKey: queryKeys.scheduling.appointments(),
    queryFn: () => apiClient.rest.get<Appointment[]>('/api/scheduling/appointments'),
  });
}

export function useAppointment(id: string) {
  return useQuery({
    queryKey: queryKeys.scheduling.appointment(id),
    queryFn: () =>
      apiClient.rest.get<Appointment>(`/api/scheduling/appointments/${id}`),
    enabled: id.length > 0,
  });
}

export function useProviderSchedule(providerId: string) {
  return useQuery({
    queryKey: queryKeys.scheduling.providerSchedule(providerId),
    queryFn: () =>
      apiClient.rest.get<ProviderSchedule>(
        `/api/scheduling/providers/${providerId}/schedule`,
      ),
    enabled: providerId.length > 0,
  });
}

export function useHoldSlot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: HoldSlotRequest) =>
      apiClient.rest.post<Appointment>('/api/scheduling/appointments/hold', input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.scheduling.appointments() });
    },
  });
}

export function useRescheduleAppointment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: RescheduleAppointmentRequest) =>
      apiClient.rest.post<Appointment>(
        `/api/scheduling/appointments/${input.appointmentId}/reschedule`,
        input,
      ),
    onSuccess: (appointment) => {
      void qc.invalidateQueries({ queryKey: queryKeys.scheduling.appointments() });
      void qc.invalidateQueries({
        queryKey: queryKeys.scheduling.appointment(appointment.id),
      });
    },
  });
}

export function useCancelAppointment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CancelAppointmentRequest) =>
      apiClient.rest.post<Appointment>(
        `/api/scheduling/appointments/${input.appointmentId}/cancel`,
        input,
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.scheduling.appointments() });
    },
  });
}

export function useRegisterWalkIn() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: RegisterWalkInRequest) =>
      apiClient.rest.post<Appointment>('/api/scheduling/appointments/walk-in', input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.scheduling.appointments() });
    },
  });
}
