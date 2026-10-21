import { StatusLabel } from '../../../components/feedback/StatusBadge';
import { useT } from '../../../i18n/locale';
import type { LinkSites } from '../../../lib/api/links';
import { useFormat } from '../../../lib/preferences/region';
import { overviewMessages } from '../messages';
import { PartFailed } from '../PartFailed';
import { useLinkSites } from '../use-overview';
import { SummaryCard, SummarySkeleton } from './SummaryCard';

/** How many old link sites there are; a port that could not open is a problem worth a line. */
function Sites({ data }: { data: LinkSites }) {
  const t = useT(overviewMessages);
  const fmt = useFormat();
  const n = data.sites.length;
  const ports = Object.keys(data.portErrors).length;
  if (n === 0) return <p className="text-sm text-kumo-subtle">{t('linksNone')}</p>;
  return (
    <>
      <p className="flex items-baseline gap-1.5">
        <span className="text-title font-semibold text-kumo-strong tabular">{fmt.number(n)}</span>
        <span className="text-sm text-kumo-subtle">{t('linksSites', { count: n })}</span>
      </p>
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
  let body;
  if (q.data) {
    body = <Sites data={q.data} />;
  } else if (q.isError) {
    body = <PartFailed message={t('cardFailed')} onRetry={() => void q.refetch()} />;
  } else {
    body = <SummarySkeleton label={t('linksTitle')} />;
  }
  return <SummaryCard title={t('linksTitle')}>{body}</SummaryCard>;
}
