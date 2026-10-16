import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useT } from '../../i18n/locale';
import { overviewKey } from '../overview/use-overview';
import { panelsMessages } from './messages';

/**
 * "Scan again" from anywhere: the overview, the panel switcher and Ctrl+K.
 * Until discovery exists (phase 3) the panel list comes with the overview,
 * so a scan means asking the server for the overview again.
 */
export function useScanPanels() {
  const t = useT(panelsMessages);
  const queryClient = useQueryClient();
  const toasts = useKumoToastManager();
  const [scanning, setScanning] = useState(false);

  async function scan() {
    setScanning(true);
    try {
      await queryClient.refetchQueries({ queryKey: overviewKey }, { throwOnError: true });
      toasts.add({ title: t('scanned') });
    } catch {
      toasts.add({ title: t('scanFailed'), variant: 'error' });
    } finally {
      setScanning(false);
    }
  }

  return { scan, scanning };
}
