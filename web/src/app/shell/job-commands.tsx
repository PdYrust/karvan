import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { FileTextIcon } from '@phosphor-icons/react';
import { PanelMark } from '../../components/panels/PanelLabel';
import { panelSearchNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import { useFormat } from '../../lib/preferences/region';
import { reportPath, reportStates } from '../../features/migrations/job-place';
import { getLinkSites, linkKeys, type LinkSite } from '../../lib/api/links';
import { listJobs, originHost, type JobRow } from '../../lib/api/migrations';
import type { CommandGroup } from './commands';
import { shellMessages } from './messages';

/** How many of the latest jobs the palette searches. */
const paletteJobs = 100;

/** The admin, the old panel's address and both panels' names, in both languages. */
function keywordsOf(j: JobRow): string {
  const kinds = [j.source.kind, j.panel.kind].map((k) => (k ? panelSearchNames[k] : ''));
  return [j.source.admin, j.source.origin, j.panel.name, ...kinds].join(' ');
}

/** A site's name, old address, domains and port, and the old panel's names. */
export function siteKeywords(s: LinkSite): string {
  return [
    s.name,
    s.sourceUrl,
    ...(s.domains ?? []),
    String(s.port),
    panelSearchNames[s.sourceKind],
  ].join(' ');
}

/**
 * The latest migrations and their reports in the palette, found by admin
 * and address, and the old link sites by name, address, domain and port. They are read only while the palette is open.
 */
export function useJobCommands(open: boolean): CommandGroup[] {
  const t = useT(shellMessages);
  const navigate = useNavigate();
  const format = useFormat();
  const jobs = useQuery({
    queryKey: ['migrations', 'palette', paletteJobs],
    queryFn: () => listJobs({ limit: paletteJobs }),
    enabled: open,
  });
  const sites = useQuery({ queryKey: linkKeys.all, queryFn: getLinkSites, enabled: open });
  const rows = jobs.data?.jobs ?? [];
  const name = (j: JobRow) => ({
    admin: j.source.admin || originHost(j.source.origin),
    host: originHost(j.source.origin),
  });
  return [
    {
      id: 'migrations',
      label: t('navMigrations'),
      searchOnly: true,
      items: rows.map((j) => ({
        id: `job:${j.id}`,
        title: t('actionJob', name(j)),
        hint: format.dateTime(new Date(j.createdAt)),
        keywords: keywordsOf(j),
        // The old panel's logo: the palette's icon slot holds one mark.
        icon: <PanelMark kind={j.source.kind} size={18} />,
        run: () => void navigate(`/migrations/${encodeURIComponent(j.id)}`),
      })),
    },
    {
      id: 'reports',
      label: t('navReports'),
      searchOnly: true,
      items: rows
        .filter((j) => reportStates.includes(j.state))
        .map((j) => ({
          id: `report:${j.id}`,
          title: t('actionReport', name(j)),
          hint: format.dateTime(new Date(j.endedAt ?? j.updatedAt)),
          keywords: keywordsOf(j),
          icon: <FileTextIcon className="size-[18px]" />,
          run: () => void navigate(reportPath(j.id)),
        })),
    },
    {
      id: 'sites',
      label: t('navLinks'),
      searchOnly: true,
      items: (sites.data?.sites ?? []).map((s) => ({
        id: `site:${String(s.id)}`,
        title: s.name,
        keywords: siteKeywords(s),
        // The old panel's logo, as on the site's card.
        icon: <PanelMark kind={s.sourceKind} size={18} />,
        run: () => void navigate(`/links/${String(s.id)}`),
      })),
    },
  ];
}
