import { ClockCounterClockwiseIcon } from '@phosphor-icons/react';
import type { UseQueryResult } from '@tanstack/react-query';
import { useState } from 'react';
import { shellMessages } from '../../../app/shell/messages';
import { recentPage, type RecentPage } from '../../../app/shell/recent-pages';
import { PanelMark } from '../../../components/panels/PanelLabel';
import { useT } from '../../../i18n/locale';
import { panelKind, type Panel } from '../../../lib/api/panels';
import { readRecents } from '../../../lib/preferences/recents';
import { HomeColumn, HomeNote, HomeRow, HomeRows, HomeRowsSkeleton } from './HomeColumn';
import { overviewMessages } from '../messages';

const sections = {
  analytics: 'navAnalytics',
  panels: 'navPanels',
  migrations: 'navMigrations',
  settings: 'settings',
} as const;

const panelOf = (page: RecentPage) =>
  page.section === 'settings' || page.section === 'migrations' ? undefined : page.panelId;

/** "Panels / logo name": the section in a quiet color, then the page. */
function RecentName({ page, panel }: { page: RecentPage; panel: Panel | undefined }) {
  const t = useT(overviewMessages);
  const ts = useT(shellMessages);
  const kind = panel ? panelKind(panel) : undefined;
  let name;
  if (page.section === 'settings') {
    name = t(page.tab === 'general' ? 'recentGeneral' : 'recentSafeMode');
  } else if (page.section === 'migrations') {
    name = t('newMigration');
  } else if (panel) {
    name = (
      <span className="inline-flex max-w-full items-center gap-1.5">
        {kind ? <PanelMark kind={kind} size={14} /> : null}
        <bdi className="truncate">{panel.name}</bdi>
      </span>
    );
  } else if (page.section === 'analytics') {
    name = t('recentServer');
  }
  return (
    <span className="flex min-w-0 items-center gap-1.5">
      <span className="shrink-0 text-kumo-subtle">{ts(sections[page.section])}</span>
      {name ? (
        <>
          <span aria-hidden="true" className="text-kumo-subtle">
            /
          </span>
          <span className="min-w-0 truncate text-kumo-strong">{name}</span>
        </>
      ) : null}
    </span>
  );
}

/** The pages this browser opened last, newest first; a panel that is gone drops out. */
export function RecentsColumn({ panels }: { panels: UseQueryResult<Panel[]> }) {
  const t = useT(overviewMessages);
  // Read once: the list only changes when the operator leaves this page.
  const [paths] = useState(readRecents);
  // Without the panel list (it failed), pages of one panel cannot be named and drop out.
  const listed = panels.data ?? [];
  const items = paths.flatMap((path) => {
    const page = recentPage(path);
    if (!page) return [];
    const id = panelOf(page);
    const panel = listed.find((p) => p.id === id);
    return id === undefined || panel ? [{ path, page, panel }] : [];
  });

  let body;
  if (paths.length > 0 && panels.isPending) {
    body = <HomeRowsSkeleton label={t('recentsTitle')} />;
  } else if (items.length === 0) {
    body = (
      <HomeNote icon={<ClockCounterClockwiseIcon size={16} aria-hidden="true" />}>
        {t('noRecents')}
      </HomeNote>
    );
  } else {
    body = (
      <HomeRows>
        {items.map(({ path, page, panel }) => (
          <HomeRow key={path} to={path}>
            <ClockCounterClockwiseIcon
              size={18}
              aria-hidden="true"
              className="shrink-0 text-kumo-subtle"
            />
            <RecentName page={page} panel={panel} />
          </HomeRow>
        ))}
      </HomeRows>
    );
  }
  return <HomeColumn title={t('recentsTitle')}>{body}</HomeColumn>;
}
