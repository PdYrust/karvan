import { cn } from '@cloudflare/kumo/utils';
import {
  CheckCircleIcon,
  CircleIcon,
  MinusCircleIcon,
  PlayCircleIcon,
  WarningCircleIcon,
  XCircleIcon,
  type Icon,
} from '@phosphor-icons/react';
import type { Translate } from '../../i18n/core';
import { useT } from '../../i18n/locale';
import type { Format } from '../../lib/format/format';
import { useFormat } from '../../lib/preferences/region';
import type { RailStep, RunStage, StepState } from './live';
import { liveMessages, type LiveKey } from './messages';

const looks: Readonly<Record<StepState, { icon: Icon; color: string }>> = {
  waiting: { icon: CircleIcon, color: 'text-kumo-subtle' },
  // Kumo's info blue is under 3:1 on white; the brand blue reads.
  running: { icon: PlayCircleIcon, color: 'text-(--color-kumo-brand)' },
  done: { icon: CheckCircleIcon, color: 'text-kumo-success' },
  failed: { icon: XCircleIcon, color: 'text-kumo-danger' },
  off: { icon: MinusCircleIcon, color: 'text-kumo-subtle' },
  notRun: { icon: MinusCircleIcon, color: 'text-kumo-subtle' },
  stopped: { icon: WarningCircleIcon, color: 'text-kumo-warning' },
};

const names: Readonly<Record<RunStage, LiveKey>> = {
  check: 'stage_check',
  backup: 'stage_backup',
  execute: 'stage_execute',
  verify: 'stage_verify',
  reload: 'stage_reload',
  undo: 'stage_undo',
  restore: 'stage_restore',
};

const words: Readonly<Record<'waiting' | 'off' | 'notRun' | 'stopped', LiveKey>> = {
  waiting: 'step_waiting',
  off: 'step_off',
  notRun: 'step_notRun',
  stopped: 'step_stopped',
};

/** How long a stage took, in words. */
function took(t: Translate<LiveKey>, fmt: Format, s: RailStep): string {
  const ms = Date.parse(s.endedAt ?? '') - Date.parse(s.startedAt ?? '');
  return ms >= 1000 ? fmt.duration(ms / 1000) : t('underSecond');
}

/** The line under a stage's name: its state in a word, with its time. */
function lineOf(t: Translate<LiveKey>, fmt: Format, s: RailStep): string {
  switch (s.state) {
    case 'done':
      return t('took', { time: took(t, fmt, s) });
    case 'failed':
      return t('failedAfter', { time: took(t, fmt, s) });
    case 'running':
      return t('runningSince', { time: fmt.time(new Date(s.startedAt ?? '')) });
    default:
      return t(words[s.state]);
  }
}

/** Check, Backup, Write, Verify, Reload, then Undo and Restore when they ran (4.6). */
export function StageRail({
  steps,
  className,
}: {
  steps: readonly RailStep[];
  className?: string;
}) {
  const t = useT(liveMessages);
  const fmt = useFormat();
  return (
    <ol
      aria-label={t('railLabel')}
      className={cn(
        'grid grid-cols-1 rounded-lg bg-kumo-base ring ring-kumo-hairline sm:auto-cols-fr sm:grid-flow-col',
        className,
      )}
    >
      {steps.map((s) => {
        const look = looks[s.state];
        const Glyph = look.icon;
        return (
          <li
            key={s.stage}
            aria-current={s.state === 'running' ? 'step' : undefined}
            className="flex min-w-0 items-start gap-3 border-b border-kumo-hairline px-4 py-3 last:border-b-0 sm:border-e sm:border-b-0 sm:last:border-e-0"
          >
            <Glyph
              size={20}
              weight={s.state === 'running' ? 'fill' : 'bold'}
              aria-hidden="true"
              className={cn('mt-0.5 shrink-0', look.color)}
            />
            <span className="flex min-w-0 flex-col">
              <span
                className={cn(
                  'font-medium',
                  s.state === 'waiting' || s.state === 'notRun' || s.state === 'off'
                    ? 'text-kumo-subtle'
                    : 'text-kumo-strong',
                )}
              >
                {t(names[s.stage])}
              </span>
              <span className="text-sm text-kumo-subtle tabular">{lineOf(t, fmt, s)}</span>
            </span>
          </li>
        );
      })}
    </ol>
  );
}
