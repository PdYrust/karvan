import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { panelsMessages } from './messages';

/**
 * Ready, or how many problems keep Karvan from using the panel. Apart from
 * PanelCells because the overview shows it too and loads with the app.
 */
export function PanelStatus({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  return panel.ready ? (
    <StatusLabel tone="success">{t('ready')}</StatusLabel>
  ) : (
    <StatusLabel tone="warning">{t('problems', { count: panel.findings.length })}</StatusLabel>
  );
}
