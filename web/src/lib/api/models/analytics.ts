/**
 * Administration & analytics bounded-context models.
 *
 * Aligned to `src/domain/administrationandanalytics` — AnalyticsDashboard and
 * ClinicDirectory aggregates. These power the admin surface's reporting and the
 * clinic/provider directory.
 */

import type { Id, IsoDateTime } from './common';

/** A single reported metric on an analytics dashboard. */
export interface DashboardMetric {
  key: string;
  label: string;
  value: number;
  /** Optional unit hint for rendering (e.g. `%`, `min`, `USD`). */
  unit?: string;
}

/** An analytics dashboard read model. */
export interface AnalyticsDashboard {
  id: Id;
  name: string;
  metrics: DashboardMetric[];
  refreshedAt: IsoDateTime;
}

/** A clinic directory entry. */
export interface ClinicDirectoryEntry {
  id: Id;
  name: string;
  active: boolean;
  providerIds: Id[];
}

/** PublishDashboardCmd: publish/refresh a dashboard snapshot. */
export interface PublishDashboardRequest {
  dashboardId: Id;
}

/** RegisterClinicCmd: add a clinic to the directory. */
export interface RegisterClinicRequest {
  name: string;
}
