import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { skipToken, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useT } from '../../i18n/locale';
import { getPanels, scanPanels, type Panel } from '../../lib/api/panels';
import { scanMessages, scanToast } from './scan-messages';

export const panelsKey = ['panels'] as const;

/**
 * Folders the last scan saw but could not register (no kind could be read),
 * with their findings. Only a scan returns them, so they are kept for the
 * rest of the visit.
 */
const scanFindsKey = ['panel-scan-finds'] as const;

/** Registered panels with a fresh look at each. The server probes every database, so it is not polled. */
export function usePanels() {
  return useQuery({ queryKey: panelsKey, queryFn: () => getPanels(), staleTime: 60_000 });
}

/** "Check again": inspects every panel and probes panel control again. */
export function useRefreshPanels() {
  const t = useT(scanMessages);
  const queryClient = useQueryClient();
  const toasts = useKumoToastManager();
  const m = useMutation({
    mutationFn: () => getPanels(true),
    onSuccess: (panels) => queryClient.setQueryData(panelsKey, panels),
    onError: () => toasts.add({ title: t('checkFailed'), variant: 'error' }),
  });
  return { refresh: () => m.mutate(), refreshing: m.isPending };
}

/** What the last scan found but could not use, if anything. */
export function useScanFinds(): Panel[] {
  return (
    useQuery<Panel[]>({
      queryKey: scanFindsKey,
      queryFn: skipToken,
      staleTime: Infinity,
      gcTime: Infinity,
    }).data ?? []
  );
}

/** "Scan this server" from anywhere: the switcher, the panels page and Ctrl+K. */
export function useScanPanels() {
  const t = useT(scanMessages);
  const queryClient = useQueryClient();
  const toasts = useKumoToastManager();
  const m = useMutation({
    mutationFn: scanPanels,
    onSuccess: (found) => {
      // Kept while no page shows them, so the panels page has them after a scan from elsewhere.
      queryClient.setQueryDefaults(scanFindsKey, { gcTime: Infinity });
      queryClient.setQueryData(
        scanFindsKey,
        found.filter((p) => p.id === 0),
      );
      toasts.add(scanToast(found, t));
      return queryClient.invalidateQueries({ queryKey: panelsKey });
    },
    onError: () => toasts.add({ title: t('scanFailed'), variant: 'error' }),
  });
  return { scan: () => m.mutate(), scanning: m.isPending };
}
