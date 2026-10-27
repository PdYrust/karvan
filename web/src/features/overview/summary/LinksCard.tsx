import { CaretRightIcon } from '@phosphor-icons/react';
import { Link } from 'react-router';
import { Bar } from '../../../components/charts/Bar';
import { StatusLabel } from '../../../components/feedback/StatusBadge';
import { useT } from '../../../i18n/locale';
import { originOf, pendingOf, type LinkSites } from '../../../lib/api/links';
import { useFormat } from '../../../lib/preferences/region';
import { overviewMessages } from '../messages';
import { PartFailed } from '../PartFailed';
import { useLinkSites } from '../use-overview';
import { SummaryCard, SummarySkeleton } from './SummaryCard';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

/** Users of every site added up. Sites of one old panel and destination share their users, so they count once. */
function totals(data: LinkSites) {
  let moved = 0;
  let verified = 0;
  let pending = 0;
  const seen = new Set<string>();
  for (const s of data.sites) {
    const key = `${String(s.panelId)} ${originOf(s.sourceUrl)}`;
    if (!s.stats || seen.has(key)) continue;
    seen.add(key);
    moved += s.stats.moved;
    verified += s.stats.verified;
    pending += pendingOf(s.stats);
  }
  return { moved, verified, pending };
}

/** How many sites, proven users against pending ones, and a port that could not open. */
function Sites({ data }: { data: LinkSites }) {
  const t = useT(overviewMessages);
  const fmt = useFormat();
  const n = data.sites.length;
  const ports = Object.keys(data.portErrors).length;
  if (n === 0) return <p className="text-sm text-kumo-subtle">{t('linksNone')}</p>;
  const u = totals(data);
  return (
    <>
      <p className="flex items-baseline gap-1.5">
        <span className="text-title font-semibold text-kumo-strong tabular">{fmt.number(n)}</span>
        <span className="text-sm text-kumo-subtle">{t('linksSites', { count: n })}</span>
      </p>
      {u.moved > 0 ? (
        // The filled part is the proven users; the rest wait for their first old link request.
        <div className="flex flex-col gap-2">
          <Bar grow share={u.verified / u.moved} />
          <p className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-kumo-subtle tabular">
            <span>{t('linksProven', { verified: u.verified, moved: u.moved })}</span>
            <span>{t('linksPending', { count: u.pending })}</span>
          </p>
        </div>
      ) : null}
      {ports > 0 ? (
        <span className="text-sm">
          <StatusLabel tone="warning">{t('linksPorts', { count: ports })}</StatusLabel>
        </span>
      ) : null}
    </>
  );
}

export function LinksCard() {
  const t = useT(overviewMessages);
  const q = useLinkSites();
  const load = useFirstLoad(q.data);
  let body;
  if (load.data) {
    body = <Sites data={load.data} />;
  } else if (q.isError) {
    body = <PartFailed message={t('cardFailed')} onRetry={() => void q.refetch()} />;
  } else {
    body = <SummarySkeleton label={t('linksTitle')} />;
  }
  return (
    <SummaryCard title={t('linksTitle')}>
      <Reveal on={load.reveal}>{body}</Reveal>
      <Link
        to="/links"
        className="mt-auto inline-flex items-center gap-1 self-start text-sm text-kumo-link hover:underline pointer-coarse:min-h-9"
      >
        {t('linksOpen')}
        <CaretRightIcon size={12} aria-hidden="true" className="rtl:-scale-x-100" />
      </Link>
    </SummaryCard>
  );
}
