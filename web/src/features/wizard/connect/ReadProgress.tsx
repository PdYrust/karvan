import { Button } from '@cloudflare/kumo/components/button';
import { useId } from 'react';
import { Bar } from '../../../components/charts/Bar';
import { useT } from '../../../i18n/locale';
import { compareStatus, type JobStream } from '../../../lib/api/events';
import { connectMessages } from './messages';

/** The live read counter of a compare, from its events. */
export function ReadProgress({
  stream,
  onStop,
  stopping,
}: {
  stream: JobStream;
  onStop: () => void;
  stopping: boolean;
}) {
  const t = useT(connectMessages);
  const id = useId();
  const building = compareStatus(stream) === 'building';
  const p = stream.progress?.stage === 'read' ? stream.progress : null;
  const line = building
    ? t('building')
    : p
      ? t(p.pass >= 2 ? 'checkingUsers' : 'readingUsers', { done: p.done, total: p.total })
      : t('readingStart');
  const share = building ? 1 : p && p.total > 0 ? p.done / p.total : 0;
  return (
    <section
      aria-labelledby={id}
      className="flex max-w-[720px] flex-col gap-4 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-6"
    >
      <h2 id={id} className="text-heading font-semibold text-kumo-strong">
        {t('readingTitle')}
      </h2>
      <div
        role="progressbar"
        aria-labelledby={id}
        aria-valuetext={line}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(share * 100)}
        className="flex flex-col gap-2"
      >
        <p aria-live="polite" className="tabular text-kumo-default">
          {line}
        </p>
        <Bar share={share} />
      </div>
      {p && p.pass >= 2 ? <p className="text-sm text-kumo-subtle">{t('readingTwice')}</p> : null}
      <div className="flex justify-end">
        <Button
          loading={stopping}
          disabled={building}
          onClick={onStop}
          className="pointer-coarse:h-11"
        >
          {t('stopReading')}
        </Button>
      </div>
    </section>
  );
}
