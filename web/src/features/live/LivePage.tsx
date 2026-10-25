import { Button } from '@cloudflare/kumo/components/button';
import { MagnifyingGlassIcon, PlayIcon } from '@phosphor-icons/react';
import { useNavigate, useParams } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import { originHost, type JobRow, type JobState } from '../../lib/api/migrations';
import { JobMove, JobStateLabel } from '../migrations/JobLabels';
import { stepPath } from '../wizard/steps';
import type { Phase } from './live';
import { LiveLog } from './LiveLog';
import { LiveScene } from './LiveScene';
import { liveMessages, type LiveKey } from './messages';
import { ResultCard } from './ResultCard';
import { StageRail } from './StageRail';
import { useLive } from './use-live';

const titles: Readonly<Record<Phase, LiveKey>> = {
  idle: 'titleLive',
  run: 'titleRun',
  undo: 'titleUndo',
  restore: 'titleRestore',
  ended: 'titleEnded',
};

type Live = ReturnType<typeof useLive>;

/** The move, the old panel's address and its admin, under the title. */
function JobLine({ row }: { row: JobRow }) {
  const t = useT(liveMessages);
  return (
    <span className="inline-flex flex-wrap items-center gap-x-3 gap-y-1">
      <JobMove job={row} />
      {row.source.origin ? <Machine>{originHost(row.source.origin)}</Machine> : null}
      {row.source.admin ? (
        <span>
          {t('admin')} <bdi className="font-medium text-kumo-default">{row.source.admin}</bdi>
        </span>
      ) : null}
    </span>
  );
}

/** The job's state for the header: running while anything runs, else how it ended. */
function stateOf(live: Live): JobState | null {
  const r = live.row.data;
  if (!r || !live.phase) return null;
  if (live.phase === 'ended') return live.end ?? r.state;
  return live.phase === 'idle' ? r.state : 'running';
}

function LiveBody({ id, live }: { id: string; live: Live }) {
  const t = useT(liveMessages);
  const navigate = useNavigate();
  const r = live.row.data;
  const { phase, ends } = live;
  if (live.row.isError) {
    return (
      <ErrorState
        message={t('loadFailed')}
        error={live.row.error}
        onRetry={() => void live.row.refetch()}
      />
    );
  }
  if (r === null) {
    return (
      <EmptyState
        icon={<MagnifyingGlassIcon size={32} />}
        title={t('notFoundTitle')}
        action={
          <Button className="pointer-coarse:h-11" onClick={() => void navigate('/')}>
            {t('toOverview')}
          </Button>
        }
      >
        {t('notFoundBody')}
      </EmptyState>
    );
  }
  if (!r || !phase || !ends) {
    return (
      <div role="status" aria-label={t('loading')} className="flex flex-col gap-6">
        <Bone className="h-16 w-full" />
        <Bone className="h-56 w-full" />
      </div>
    );
  }
  if (phase === 'idle') {
    return (
      <EmptyState
        icon={<PlayIcon size={32} />}
        title={t('idleTitle')}
        action={
          <Button
            className="pointer-coarse:h-11"
            onClick={() => void navigate(stepPath('start', id))}
          >
            {t('openStart')}
          </Button>
        }
      >
        {t('idleBody')}
      </EmptyState>
    );
  }
  return (
    <>
      {live.lost && phase !== 'ended' ? (
        <InlineAlert tone="warning" className="order-first">
          {t('lost')}
        </InlineAlert>
      ) : null}
      {phase === 'ended' ? (
        <ResultCard
          jobId={id}
          state={live.end ?? r.state}
          row={r}
          report={live.report.data}
          reportLoading={live.report.isLoading}
        />
      ) : (
        <LiveScene
          jobId={id}
          phase={phase}
          stream={live.stream}
          counter={live.counter}
          skipped={live.skipped}
          ends={ends}
          backupId={live.report.data?.backupId ?? ''}
          safeMode={r.safeMode}
          stoppable={live.stoppable}
        />
      )}
      {/* On a phone what happens now comes first; from 640 px the rail heads the page. */}
      <StageRail steps={live.rail} className="sm:order-first" />
    </>
  );
}

/** Where Start leads (4.6): the stage rail, the live scene, the log, and the result once it ends. */
export function LivePage() {
  const { id = '' } = useParams();
  const t = useT(liveMessages);
  const live = useLive(id);
  const r = live.row.data;
  const title = t(live.phase ? titles[live.phase] : 'titleLive');
  usePageTitle(title);
  const state = stateOf(live);
  const withLog = Boolean(r && live.phase && live.phase !== 'idle');
  return (
    <div className="flex flex-col">
      <PageHeader
        eyebrow={t('eyebrow')}
        title={title}
        description={r ? <JobLine row={r} /> : undefined}
        actions={state ? <JobStateLabel state={state} /> : undefined}
      />
      <div
        className={
          withLog ? 'grid max-w-[1400px] lg:grid-cols-[minmax(0,1fr)_22rem]' : 'max-w-[1400px]'
        }
      >
        <div className="flex min-w-0 flex-col gap-6 p-4 sm:p-8">
          <LiveBody id={id} live={live} />
        </div>
        {withLog ? (
          <aside className="border-t border-kumo-hairline p-4 sm:p-8 lg:border-s lg:border-t-0">
            <LiveLog events={live.stream.events} />
          </aside>
        ) : null}
      </div>
    </div>
  );
}
