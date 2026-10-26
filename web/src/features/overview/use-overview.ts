import { useQuery } from '@tanstack/react-query';
import { getLinkSites, linkKeys } from '../../lib/api/links';
import { listJobs } from '../../lib/api/migrations';
import type { Panel } from '../../lib/api/panels';
import { getOverview } from '../../lib/api/overview';
import { backupKeys, getBackups } from '../../lib/api/safemode';

export const overviewKey = ['overview'] as const;

/** Version, Safe Mode, account security and the migrations summed up. */
export function useOverview() {
  return useQuery({ queryKey: overviewKey, queryFn: getOverview });
}

/** How many rows each of the overview's three columns shows. */
export const homeRowsMax = 3;
/** How many jobs the Migrations column shows. */
export const recentJobsLimit = homeRowsMax;
/** The panels the Panels column shows: the last ones registered, in the list's order. */
export function homePanels(panels: readonly Panel[]): Panel[] {
  return panels.filter((p) => p.id > 0).slice(-homeRowsMax);
}

export function useRecentJobs() {
  return useQuery({
    queryKey: ['migrations', 'recent', recentJobsLimit],
    queryFn: () => listJobs({ limit: recentJobsLimit }),
  });
}

export function useBackups() {
  return useQuery({ queryKey: backupKeys.all, queryFn: getBackups });
}

export function useLinkSites() {
  return useQuery({ queryKey: linkKeys.all, queryFn: getLinkSites });
}
