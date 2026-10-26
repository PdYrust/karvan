import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  getPanelStats,
  getPanelsStats,
  getServerStats,
  type RangeKey,
  type ServerStats,
} from '../../lib/api/analytics';
import { databaseRange } from './range';
import { mergeLive } from './server-view';

export const analyticsKey = ['analytics'] as const;

const serverKey = (r: RangeKey) => [...analyticsKey, 'server', r] as const;

// The server samples every 5 seconds and keeps minutes; panel databases
// are read once a minute. Polling pauses while the tab is hidden.
const liveEvery = 5_000;
const minutesEvery = 60_000;
const databasesEvery = 30_000;

/** This server's numbers; the live range adds each poll's new samples to the ones it has. */
export function useServerStats(range: RangeKey) {
  const qc = useQueryClient();
  return useQuery({
    queryKey: serverKey(range),
    queryFn: async () => {
      if (range !== 'live') return getServerStats(range);
      const prev = qc.getQueryData<ServerStats>(serverKey('live'));
      return mergeLive(prev, await getServerStats('live', prev?.seq));
    },
    refetchInterval: range === 'live' ? liveEvery : minutesEvery,
  });
}

/** One panel database in full: reading, minutes, tables, hourly traffic, sign-ups, apps. */
export function usePanelStats(id: number, range: RangeKey) {
  const r = databaseRange(range);
  return useQuery({
    queryKey: [...analyticsKey, 'panel', id, r],
    queryFn: () => getPanelStats(id, r),
    refetchInterval: databasesEvery,
    enabled: id > 0,
  });
}

/**
 * Registered panels for the tabs. The analytics list answers from memory,
 * while the panel list looks at every panel first and can take seconds.
 */
export function usePanelTabs() {
  return useQuery({
    queryKey: [...analyticsKey, 'panels'],
    queryFn: () => getPanelsStats('1h'),
    staleTime: 60_000,
    select: (d) => d.panels,
  });
}
