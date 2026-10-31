import { HourglassIcon } from '@phosphor-icons/react';
import { useId, useState } from 'react';
import { CompactCard, CompactCardSkeleton } from '../../../components/charts/CompactCard';
import { Reveal } from '../../../components/feedback/Reveal';
import { FactList } from '../../../components/layout/FactList';
import { useT } from '../../../i18n/locale';
import type { Sample, ServerStats } from '../../../lib/api/analytics';
import { useFirstLoad } from '../../../lib/motion/use-first-load';
import { analyticsMessages } from '../../analytics/messages';
import { useServerFacts } from '../../analytics/server/FactsCard';
import { LoadFailed, Notice, noticeIcon } from '../../analytics/states';
import { useServerStats } from '../../analytics/use-analytics';
import { overviewMessages } from '../messages';
import { CardDetail } from './CardDetail';
import { OverviewCard } from './OverviewCard';
import { useServerCards } from './server-cards';

const grid = 'grid gap-4 lg:grid-cols-2 2xl:grid-charts';

/** What the server is: facts that show everything, so the card takes no press. */
function FactsCard({ stats, now }: { stats: ServerStats; now: Sample }) {
  const t = useT(analyticsMessages);
  return (
    <CompactCard title={t('factsTitle')} within={t('in_live')}>
      <FactList facts={useServerFacts(stats, now)} />
    </CompactCard>
  );
}

/**
 * This server as compact cards, always live, like the dashboard home's
 * analytics. The panel databases live on the analytics pages.
 */
export function OverviewAnalytics() {
  const t = useT(overviewMessages);
  const ta = useT(analyticsMessages);
  const id = useId();
  const q = useServerStats('live');
  const load = useFirstLoad(q.data);
  const server = useServerCards(load.data);
  const [open, setOpen] = useState(false);
  const [shown, setShown] = useState<string>();
  let body;
  if (server && load.data?.now) {
    const { cards, view, within } = server;
    body = (
      <div className={grid}>
        {cards.map((c) => (
          <OverviewCard
            key={c.id}
            card={c}
            view={view}
            within={within}
            onOpen={() => {
              setShown(c.id);
              setOpen(true);
            }}
          />
        ))}
        <FactsCard stats={load.data} now={load.data.now} />
      </div>
    );
  } else if (load.data) {
    body = <Notice icon={<HourglassIcon {...noticeIcon} />}>{ta('serverWaiting')}</Notice>;
  } else if (q.isError) {
    body = <LoadFailed error={q.error} onRetry={() => void q.refetch()} />;
  } else {
    body = (
      <div role="status" aria-label={t('analyticsTitle')} className={grid}>
        {[0, 1, 2, 3].map((i) => (
          <CompactCardSkeleton key={i} tall />
        ))}
      </div>
    );
  }
  return (
    <section aria-labelledby={id} className="flex flex-col gap-4">
      <h2 id={id} className="text-heading font-semibold text-kumo-strong">
        {t('analyticsTitle')}
      </h2>
      <Reveal on={load.reveal}>{body}</Reveal>
      <CardDetail
        card={server?.cards.find((c) => c.id === shown)}
        view={server?.view}
        within={server?.within ?? ''}
        open={open}
        onOpenChange={setOpen}
      />
    </section>
  );
}
