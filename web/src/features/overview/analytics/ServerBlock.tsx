import { HourglassIcon } from '@phosphor-icons/react';
import { useId } from 'react';
import { useT } from '../../../i18n/locale';
import type { RangeKey } from '../../../lib/api/analytics';
import { analyticsMessages } from '../../analytics/messages';
import { rangeSpan } from '../../analytics/range';
import { ServerTiles } from '../../analytics/server/ServerTiles';
import { serverView } from '../../analytics/server-view';
import { LoadFailed, Notice, noticeIcon } from '../../analytics/states';
import { useServerStats } from '../../analytics/use-analytics';
import { overviewMessages } from '../messages';
import { BlockHeading, TilesSkeleton } from './BlockHeading';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

/** This server's CPU, memory, disk and network, each with its trend over the range. */
export function ServerBlock({ range, search }: { range: RangeKey; search: string }) {
  const t = useT(overviewMessages);
  const ta = useT(analyticsMessages);
  const id = useId();
  const q = useServerStats(range);
  const load = useFirstLoad(q.data);
  let body;
  if (!load.data) {
    body = q.isError ? (
      <LoadFailed error={q.error} onRetry={() => void q.refetch()} />
    ) : (
      <TilesSkeleton label={t('thisServer')} />
    );
  } else if (!load.data.now) {
    body = <Notice icon={<HourglassIcon {...noticeIcon} />}>{ta('serverWaiting')}</Notice>;
  } else {
    body = (
      <ServerTiles
        stats={load.data}
        now={load.data.now}
        view={serverView(load.data, rangeSpan[range])}
      />
    );
  }
  return (
    <section aria-labelledby={id} className="flex flex-col gap-3">
      <BlockHeading id={id} to={`/analytics/server${search}`}>
        {t('thisServer')}
      </BlockHeading>
      <Reveal on={load.reveal}>{body}</Reveal>
    </section>
  );
}
