import { cn } from '@cloudflare/kumo/utils';
import { HardDrivesIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { cardLink, RowCard } from '../../components/data/RowCard';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { PanelMark } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import { databaseOf, engineName, panelKind, type Panel } from '../../lib/api/panels';
import { controlMessages, controlText } from './control';
import { panelsMessages } from './messages';
import { CurrentTag, PanelVersion } from './PanelCells';
import { PanelStatus } from './PanelStatus';

/** The logo and the folder name, the card's link to the panel's page. */
function CardName({ panel, current }: { panel: Panel; current: boolean }) {
  const kind = panelKind(panel);
  return (
    <span className="flex min-w-0 items-center gap-2">
      {kind ? (
        <PanelMark kind={kind} size={20} />
      ) : (
        <HardDrivesIcon size={20} aria-hidden="true" className="shrink-0 text-kumo-subtle" />
      )}
      {/* dir="auto" isolates the name and cuts a Latin one at its end in Persian too. */}
      <Link
        to={`/panels/${String(panel.id)}`}
        dir="auto"
        className={cn('min-w-0 truncate font-medium text-kumo-strong', cardLink)}
      >
        {panel.name}
      </Link>
      {current ? <CurrentTag /> : null}
    </span>
  );
}

/** One status line: Ready or the problems, and panel control only when Karvan can't use it. */
function CardStatus({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  const tc = useT(controlMessages);
  const control = panel.control;
  return (
    <p className="flex flex-wrap items-center gap-x-3 gap-y-1">
      <PanelStatus panel={panel} />
      {control && !control.available ? (
        <StatusLabel tone="warning">
          {t('colControl')}: {tc(controlText(control).label)}
        </StatusLabel>
      ) : null}
    </p>
  );
}

/**
 * On a phone a panel is a row card like a certificate's: the logo, the name
 * and the menu, one quiet line with the version and the database, then the
 * status. The database's address is on the panel's page.
 */
export function PanelCard({
  panel,
  current,
  menu,
}: {
  panel: Panel;
  current: boolean;
  menu: ReactNode;
}) {
  const t = useT(panelsMessages);
  const { engine } = databaseOf(panel);
  return (
    <RowCard lead={<CardName panel={panel} current={current} />} menu={menu}>
      <div className="flex flex-col gap-2 text-sm">
        <p className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-kumo-subtle">
          <PanelVersion panel={panel} />
          <span>{engine ? engineName(engine) : t('notRead')}</span>
        </p>
        <CardStatus panel={panel} />
      </div>
    </RowCard>
  );
}
