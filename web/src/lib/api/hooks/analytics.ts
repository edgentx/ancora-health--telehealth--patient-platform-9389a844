/**
 * Administration & analytics hooks (dashboards, clinic directory).
 *
 * Publishing a dashboard or registering a clinic invalidates the corresponding
 * list so the admin surface reflects the change without a manual refresh.
 */
'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '../client';
import type {
  AnalyticsDashboard,
  ClinicDirectoryEntry,
  PublishDashboardRequest,
  RegisterClinicRequest,
} from '../models/analytics';
import { queryKeys } from './keys';

export function useDashboards() {
  return useQuery({
    queryKey: queryKeys.analytics.dashboards(),
    queryFn: () =>
      apiClient.rest.get<AnalyticsDashboard[]>('/api/analytics/dashboards'),
  });
}

export function useDashboard(id: string) {
  return useQuery({
    queryKey: queryKeys.analytics.dashboard(id),
    queryFn: () =>
      apiClient.rest.get<AnalyticsDashboard>(`/api/analytics/dashboards/${id}`),
    enabled: id.length > 0,
  });
}

export function useClinics() {
  return useQuery({
    queryKey: queryKeys.analytics.clinics(),
    queryFn: () =>
      apiClient.rest.get<ClinicDirectoryEntry[]>('/api/analytics/clinics'),
  });
}

export function usePublishDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: PublishDashboardRequest) =>
      apiClient.rest.post<AnalyticsDashboard>(
        `/api/analytics/dashboards/${input.dashboardId}/publish`,
        input,
      ),
    onSuccess: (dashboard) => {
      void qc.invalidateQueries({ queryKey: queryKeys.analytics.dashboards() });
      void qc.invalidateQueries({
        queryKey: queryKeys.analytics.dashboard(dashboard.id),
      });
    },
  });
}

export function useRegisterClinic() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: RegisterClinicRequest) =>
      apiClient.rest.post<ClinicDirectoryEntry>('/api/analytics/clinics', input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.analytics.clinics() });
    },
  });
}
