import { ShieldCheckIcon } from '@phosphor-icons/react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { PanelMark } from '../../components/panels/PanelLabel';
import { panelSearchNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import { useFormat } from '../../lib/preferences/region';
import { batchKeys, listBatches, type Batch } from '../../lib/api/batches';
import { certificateKeys, getCertificates, type Certificate } from '../../lib/api/certificates';
import { getLinkSites, linkKeys, type LinkSite } from '../../lib/api/rules';
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

/** A batch's name, admins, old address and both panels' names. */
export function batchKeywords(b: Batch): string {
  const kinds = [b.source.kind, b.panel.kind].map((k) => (k ? panelSearchNames[k] : ''));
  return [
    b.name,
    ...b.admins.flatMap((a) => [a.source, a.dest]),
    b.source.origin,
    b.panel.name,
    ...kinds,
  ].join(' ');
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

/** A certificate's name, the names it covers and its files. */
export function certificateKeywords(c: Certificate): string {
  return [c.name, ...c.facts.domains, c.certFile ?? '', c.keyFile ?? ''].join(' ');
}

/**
 * The latest migrations and batches in the palette, found by admin, name
 * and address, the old link sites by name, address, domain and port, and the
 * certificates by name, domain and file. They are read only while the palette is open.
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
  const batches = useQuery({
    queryKey: batchKeys.list({ limit: paletteJobs }),
    queryFn: () => listBatches({ limit: paletteJobs }),
    enabled: open,
  });
  const sites = useQuery({ queryKey: linkKeys.all, queryFn: getLinkSites, enabled: open });
  const certs = useQuery({
    queryKey: certificateKeys.all,
    queryFn: getCertificates,
    enabled: open,
  });
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
      id: 'batches',
      label: t('navBatches'),
      searchOnly: true,
      items: (batches.data?.batches ?? []).map((b) => ({
        id: `batch:${String(b.id)}`,
        title: b.name,
        hint: format.dateTime(new Date(b.createdAt)),
        keywords: batchKeywords(b),
        icon: <PanelMark kind={b.source.kind} size={18} />,
        run: () => void navigate(`/batches/${String(b.id)}`),
      })),
    },
    {
      id: 'sites',
      label: t('navRules'),
      searchOnly: true,
      items: (sites.data?.sites ?? []).map((s) => ({
        id: `site:${String(s.id)}`,
        title: s.name,
        keywords: siteKeywords(s),
        // The old panel's logo, as on the site's card.
        icon: <PanelMark kind={s.sourceKind} size={18} />,
        run: () => void navigate(`/rules/${String(s.id)}`),
      })),
    },
    {
      id: 'certificates',
      label: t('navDomains'),
      searchOnly: true,
      items: (certs.data?.certificates ?? []).map((c) => ({
        id: `certificate:${String(c.id)}`,
        title: c.name,
        keywords: certificateKeywords(c),
        icon: <ShieldCheckIcon className="size-[18px]" />,
        // Let's Encrypt ones renew themselves and have no form.
        run: () =>
          void navigate(c.origin === 'acme' ? '/domains' : `/domains/certificates/${String(c.id)}`),
      })),
    },
  ];
}
