import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useT } from '../../i18n/locale';
import { getPanels, scanPanels } from '../../lib/api/panels';
import { panelsMessages } from './messages';

export const panelsKey = ['panels'] as const;

/** Registered panels with a fresh look at each. The server probes every database, so it is not polled. */
export function usePanels() {
  return useQuery({ queryKey: panelsKey, queryFn: () => getPanels(), staleTime: 60_000 });
}

/** "Scan this server" from anywhere: the switcher, the panels page and Ctrl+K. */
export function useScanPanels() {
  const t = useT(panelsMessages);
  const queryClient = useQueryClient();
  const toasts = useKumoToastManager();
  const m = useMutation({
    mutationFn: scanPanels,
    onSuccess: (found) => {
      const usable = found.filter((p) => p.id > 0).length;
      toasts.add({
        title: usable === 0 ? t('scanNone') : t('scanFound', { count: usable }),
      });
      return queryClient.invalidateQueries({ queryKey: panelsKey });
    },
    onError: () => toasts.add({ title: t('scanFailed'), variant: 'error' }),
  });
  return { scan: () => m.mutate(), scanning: m.isPending };
}
