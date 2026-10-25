import { useId } from 'react';
import { useT } from '../../i18n/locale';
import type { MigrationEvent } from '../../lib/api/events';
import { errorMessages } from '../../lib/errors/errors';
import { useFormat } from '../../lib/preferences/region';
import { eventSentence } from './log';
import { liveMessages } from './messages';

/**
 * The readable log (4.6): one plain sentence per stage, log line and end,
 * with its time to the second in the operator's zone, oldest first. New
 * lines are read out politely.
 */
export function LiveLog({ events }: { events: readonly MigrationEvent[] }) {
  const t = useT(liveMessages);
  const te = useT(errorMessages);
  const fmt = useFormat();
  const id = useId();
  const lines = events.flatMap((e) => {
    const s = eventSentence(e);
    if (!s) return [];
    return [{ seq: e.seq, at: e.at, text: s.from === 'live' ? t(s.key, s.params) : te(s.key) }];
  });
  return (
    <section aria-labelledby={id} className="flex flex-col gap-2">
      <h2 id={id} className="text-heading font-semibold text-kumo-strong">
        {t('logTitle')}
      </h2>
      {lines.length === 0 ? (
        <p className="text-kumo-subtle">{t('logEmpty')}</p>
      ) : (
        <div role="log">
          <ol className="flex flex-col">
            {lines.map((l) => (
              <li
                key={l.seq}
                className="flex items-baseline gap-3 border-b border-kumo-hairline py-2.5 last:border-b-0"
              >
                <time dateTime={l.at} className="shrink-0 text-sm text-kumo-subtle tabular">
                  {fmt.seconds(new Date(l.at))}
                </time>
                <span className="min-w-0 text-kumo-default">{l.text}</span>
              </li>
            ))}
          </ol>
        </div>
      )}
    </section>
  );
}
