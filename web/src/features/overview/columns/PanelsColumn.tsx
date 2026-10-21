import { HardDrivesIcon } from '@phosphor-icons/react';
import type { UseQueryResult } from '@tanstack/react-query';
import { StatusLabel } from '../../../components/feedback/StatusBadge';
import { PanelMark } from '../../../components/panels/PanelLabel';
import { useT } from '../../../i18n/locale';
import { panelKind, type Panel } from '../../../lib/api/panels';
import { panelsMessages } from '../../panels/messages';
import { PartFailed } from '../PartFailed';
import { HomeColumn, HomeRow, HomeRows, HomeRowsSkeleton } from './HomeColumn';
import { overviewMessages } from '../messages';

function PanelRow({ panel }: { panel: Panel }) {
  const tp = useT(panelsMessages);
  const kind = panelKind(panel);
  return (
    <HomeRow to={`/panels/${String(panel.id)}`}>
      {kind ? (
        <PanelMark kind={kind} size={18} />
      ) : (
        <HardDrivesIcon size={18} aria-hidden="true" className="shrink-0 text-kumo-subtle" />
      )}
      <bdi className="min-w-0 truncate font-medium text-kumo-strong">{panel.name}</bdi>
      <span className="ms-auto shrink-0 text-sm">
        {panel.ready ? (
          <StatusLabel tone="success">{tp('ready')}</StatusLabel>
        ) : (
          <StatusLabel tone="warning">
            {tp('problems', { count: panel.findings.length })}
          </StatusLabel>
        )}
      </span>
    </HomeRow>
  );
}

/** Each registered panel and whether Karvan can work with it. */
export function PanelsColumn({ panels }: { panels: UseQueryResult<Panel[]> }) {
  const t = useT(overviewMessages);
  let body;
  if (panels.data) {
    body = (
      <HomeRows>
        {panels.data
          .filter((p) => p.id > 0)
          .map((p) => (
            <PanelRow key={p.id} panel={p} />
          ))}
      </HomeRows>
    );
  } else if (panels.isError) {
    body = <PartFailed message={t('listFailed')} onRetry={() => void panels.refetch()} />;
  } else {
    body = <HomeRowsSkeleton label={t('panelsTitle')} />;
  }
  return (
    <HomeColumn title={t('panelsTitle')} to="/panels">
      {body}
    </HomeColumn>
  );
}
