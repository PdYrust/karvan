import { cn } from '@cloudflare/kumo/utils';
import { CheckIcon } from '@phosphor-icons/react';
import { Link } from 'react-router';
import { useFormat } from '../../lib/preferences/region';
import { Bar } from '../charts/Bar';

export interface Step {
  label: string;
  /** Set on a step the operator may go back to. */
  to?: string;
}

const ring = 'grid size-6 shrink-0 place-items-center rounded-full text-sm tabular ring';

function Marker({ index, done, current }: { index: number; done: boolean; current: boolean }) {
  const fmt = useFormat();
  if (done) {
    return (
      <span className={cn(ring, 'bg-kumo-tint text-kumo-default ring-kumo-line')}>
        <CheckIcon size={14} weight="bold" aria-hidden="true" />
      </span>
    );
  }
  return (
    <span
      aria-hidden="true"
      className={cn(
        ring,
        current ? 'bg-kumo-brand text-on-brand ring-kumo-brand' : 'text-kumo-subtle ring-kumo-line',
      )}
    >
      {fmt.number(index + 1)}
    </span>
  );
}

/**
 * Where a wizard stands. From 640 px every step in one row (done steps are
 * links back); on phones "Step 2 of 6: Summary" over a thin bar.
 */
export function Steps({
  steps,
  current,
  label,
  progress,
}: {
  steps: Step[];
  current: number;
  /** Names the list for screen readers, such as "Migration steps". */
  label: string;
  /** The phone line, such as "Step 2 of 6: Summary". */
  progress: string;
}) {
  return (
    <nav aria-label={label}>
      <ol className="hidden flex-wrap items-center gap-x-2 gap-y-3 sm:flex">
        {steps.map((s, i) => {
          const done = i < current;
          const isCurrent = i === current;
          const body = (
            <>
              <Marker index={i} done={done} current={isCurrent} />
              <span
                className={cn(
                  isCurrent
                    ? 'font-medium text-(--color-kumo-brand)'
                    : done
                      ? 'text-kumo-default'
                      : 'text-kumo-subtle',
                )}
              >
                {s.label}
              </span>
            </>
          );
          return (
            <li key={s.label} className="flex items-center gap-2">
              {done && s.to ? (
                <Link
                  to={s.to}
                  className="inline-flex min-h-11 items-center gap-2 rounded-md underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-kumo-brand"
                >
                  {body}
                </Link>
              ) : (
                <span
                  className="inline-flex min-h-11 items-center gap-2"
                  aria-current={isCurrent ? 'step' : undefined}
                >
                  {body}
                </span>
              )}
              {i < steps.length - 1 ? (
                <span aria-hidden="true" className="h-px w-6 bg-kumo-line lg:w-10" />
              ) : null}
            </li>
          );
        })}
      </ol>
      <div className="flex flex-col gap-2 sm:hidden">
        <p aria-current="step" className="text-sm font-medium text-kumo-strong">
          {progress}
        </p>
        <Bar share={(current + 1) / steps.length} />
      </div>
    </nav>
  );
}
