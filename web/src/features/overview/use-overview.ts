import { useQuery } from '@tanstack/react-query';
import { getLinkSites, linkKeys } from '../../lib/api/links';
import { listJobs } from '../../lib/api/migrations';
import { getOverview } from '../../lib/api/overview';
import { backupKeys, getBackups } from '../../lib/api/safemode';

export const overviewKey = ['overview'] as const;

/** Version, Safe Mode, account security and the migrations summed up. */
export function useOverview() {
  return useQuery({ queryKey: overviewKey, queryFn: getOverview });
}

/** How many jobs the Migrations column shows. */
export const recentJobsLimit = 5;

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
