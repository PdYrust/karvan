import { HourglassIcon } from '@phosphor-icons/react';
import { useId } from 'react';
import { PanelMark } from '../../../components/panels/PanelLabel';
import { useT } from '../../../i18n/locale';
import type { PanelAnalytics, RangeKey } from '../../../lib/api/analytics';
import type { PanelKind } from '../../../lib/api/panels';
import { CardHeadings } from '../../analytics/ChartCard';
import { TrafficCard } from '../../analytics/databases/HistoryCards';
import { StatusCard } from '../../analytics/databases/UsersCards';
import { panelWindow } from '../../analytics/databases/window';
import { analyticsMessages } from '../../analytics/messages';
import { DatabaseProblem, LoadFailed, Notice, noticeIcon } from '../../analytics/states';
import { usePanelStats } from '../../analytics/use-analytics';
import { BlockHeading, TilesSkeleton } from './BlockHeading';
import { DatabaseTiles } from './DatabaseTiles';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

/** Tiles, users by status and traffic per hour, once the database has been read. */
function Numbers({
  stats,
  range,
  of,
}: {
  stats: PanelAnalytics;
  range: RangeKey;
  /** The block heading's id. */
  of: string;
}) {
  const t = useT(analyticsMessages);
  const { panel, detail, detailProblem } = stats;
  if (panel.problem) return <DatabaseProblem code={panel.problem.code} />;
  if (panel.pending || !panel.data) {
    return <Notice icon={<HourglassIcon {...noticeIcon} />}>{t('pending')}</Notice>;
  }
  const { end, step, domain } = panelWindow(stats, range);
  const missing =
    detailProblem === undefined
      ? undefined
      : t(detailProblem === 'pending' ? 'pending' : 'detailUnreachable');
  return (
    <>
      <DatabaseTiles panel={panel} data={panel.data} step={step} domain={domain} />
      <div className="grid gap-4 lg:grid-cols-2">
        <CardHeadings level={4} of={of}>
          <StatusCard data={panel.data} />
          <TrafficCard detail={detail} missing={missing} end={end} range={range} />
        </CardHeadings>
      </div>
    </>
  );
}

/** One panel database on the overview, named with its logo and linking to its own analytics. */
export function DatabaseBlock({
  panelId,
  name,
  kind,
  range,
  search,
}: {
  panelId: number;
  name: string;
  kind: PanelKind | undefined;
  range: RangeKey;
  search: string;
}) {
  const t = useT(analyticsMessages);
  const id = useId();
  const q = usePanelStats(panelId, range);
  const load = useFirstLoad(q.data);
  let body;
  if (load.data) {
    body = <Numbers stats={load.data} range={range} of={id} />;
  } else if (q.isError) {
    body = <LoadFailed error={q.error} onRetry={() => void q.refetch()} />;
  } else {
    body = <TilesSkeleton label={name || t('title')} />;
  }
  return (
    <section aria-labelledby={id} className="flex flex-col gap-3">
      <BlockHeading id={id} to={`/analytics/panels/${String(panelId)}${search}`}>
        {kind ? <PanelMark kind={kind} size={18} /> : null}
        <bdi className="truncate">{name}</bdi>
      </BlockHeading>
      <Reveal on={load.reveal}>{body}</Reveal>
    </section>
  );
}
