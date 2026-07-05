/**
 * Clinical query/mutation hooks (encounters, lab orders).
 *
 * Mutations invalidate the encounter list plus the affected detail key so a
 * signed note or placed order is reflected everywhere it is shown.
 */
'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '../client';
import type {
  DocumentEncounterRequest,
  Encounter,
  LabOrder,
  PlaceLabOrderRequest,
  SignEncounterRequest,
} from '../models/clinical';
import { queryKeys } from './keys';

export function useEncounters() {
  return useQuery({
    queryKey: queryKeys.clinical.encounters(),
    queryFn: () => apiClient.rest.get<Encounter[]>('/api/clinical/encounters'),
  });
}

export function useEncounter(id: string) {
  return useQuery({
    queryKey: queryKeys.clinical.encounter(id),
    queryFn: () => apiClient.rest.get<Encounter>(`/api/clinical/encounters/${id}`),
    enabled: id.length > 0,
  });
}

export function useLabOrders(encounterId: string) {
  return useQuery({
    queryKey: queryKeys.clinical.labOrders(encounterId),
    queryFn: () =>
      apiClient.rest.get<LabOrder[]>(
        `/api/clinical/encounters/${encounterId}/lab-orders`,
      ),
    enabled: encounterId.length > 0,
  });
}

export function useDocumentEncounter() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: DocumentEncounterRequest) =>
      apiClient.rest.post<Encounter>(
        `/api/clinical/encounters/${input.encounterId}/document`,
        input,
      ),
    onSuccess: (encounter) => {
      void qc.invalidateQueries({
        queryKey: queryKeys.clinical.encounter(encounter.id),
      });
    },
  });
}

export function useSignEncounter() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: SignEncounterRequest) =>
      apiClient.rest.post<Encounter>(
        `/api/clinical/encounters/${input.encounterId}/sign`,
        input,
      ),
    onSuccess: (encounter) => {
      void qc.invalidateQueries({ queryKey: queryKeys.clinical.encounters() });
      void qc.invalidateQueries({
        queryKey: queryKeys.clinical.encounter(encounter.id),
      });
    },
  });
}

export function usePlaceLabOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: PlaceLabOrderRequest) =>
      apiClient.rest.post<LabOrder>('/api/clinical/lab-orders', input),
    onSuccess: (order) => {
      void qc.invalidateQueries({
        queryKey: queryKeys.clinical.labOrders(order.encounterId),
      });
    },
  });
}
