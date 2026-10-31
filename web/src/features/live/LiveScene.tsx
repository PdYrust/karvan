import { useEffect, useState } from 'react';
import { Bar } from '../../components/charts/Bar';
import { useT } from '../../i18n/locale';
import type { JobStream } from '../../lib/api/events';
import { useFormat } from '../../lib/preferences/region';
import { FlowScene, type SceneEnds } from './FlowScene';
import { panelBack, type Counter, type Phase } from './live';
import { eventSentence } from './log';
import { liveMessages } from './messages';
import { RewindScene } from './RewindScene';
import { StopButton } from './StopButton';
import { useCalm } from './use-calm';

/** Seconds since a time, ticking once a second while on. */
function useSince(from: string | undefined, on: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!on) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [on]);
  return from ? Math.max(0, (now - Date.parse(from)) / 1000) : 0;
}

/** What the drawing's tiles say, as plain text in place of it: nothing moves. */
function CalmNumbers({ counter, skipped }: { counter: Counter | null; skipped: number }) {
  const t = useT(liveMessages);
  if (counter?.kind !== 'move') return null;
  return (
    <p className="flex flex-wrap gap-x-4 text-sm text-kumo-subtle tabular">
      <span>{t('toGo', { count: Math.max(0, counter.total - counter.done) })}</span>
      {skipped > 0 ? <span>{t('skippedCount', { count: skipped })}</span> : null}
    </p>
  );
}

export interface SceneProps {
  jobId: string;
  phase: Phase;
  stream: JobStream;
  counter: Counter | null;
  skipped: number;
  ends: SceneEnds;
  backupId: string;
  safeMode: boolean;
  stoppable: boolean;
}

/**
 * The live card: what happens now in heading size, the server's
 * estimate, Stop, a meter, and the scene of the phase; with less motion
 * asked for, or Data Saver on, the meter and the numbers alone.
 */
export function LiveScene(p: SceneProps) {
  const t = useT(liveMessages);
  const fmt = useFormat();
  const calm = useCalm();
  const restoring = p.phase === 'restore';
  const restore = p.stream.stages.findLast((s) => s.stage === 'restore');
  const online = restoring && panelBack(p.stream);
  const offline = useSince(restore?.startedAt, restoring && !online);
  const c = p.counter;
  const now = p.stream.events.findLast((e) => e.type === 'stage' && e.data.state === 'start');
  const said = now ? eventSentence(now) : null;
  const line = restoring
    ? online
      ? t('start_restore')
      : t('offlineFor', { time: fmt.clock(offline) })
    : c
      ? t(c.kind === 'move' ? 'moving' : 'removing', { done: c.done, total: c.total })
      : said?.from === 'live'
        ? t(said.key)
        : t('titleLive');
  return (
    <section
      aria-label={t('progress')}
      className="flex flex-col gap-5 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-6"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <p className="text-heading font-semibold text-kumo-strong tabular">{line}</p>
          {c && c.etaSeconds > 0 ? (
            <p className="text-sm text-kumo-subtle">
              {t('eta', { time: fmt.duration(c.etaSeconds, 1) })}
            </p>
          ) : null}
        </div>
        {p.stoppable ? <StopButton jobId={p.jobId} safeMode={p.safeMode} /> : null}
      </div>
      {c ? <Bar share={c.total > 0 ? c.done / c.total : 0} className="h-2" /> : null}
      {calm ? (
        <CalmNumbers counter={c} skipped={p.skipped} />
      ) : restoring ? (
        <RewindScene ends={p.ends} backupId={p.backupId} online={online} />
      ) : (
        <FlowScene
          mode={p.phase === 'undo' ? 'undo' : 'move'}
          counter={c}
          skipped={p.phase === 'undo' ? 0 : p.skipped}
          ends={p.ends}
        />
      )}
    </section>
  );
}
