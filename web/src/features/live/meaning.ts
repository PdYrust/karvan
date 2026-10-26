import { useT } from '../../i18n/locale';
import { reloadOutcome, type ReloadOutcome, type Report } from '../../lib/api/report';
import { errorMessages } from '../../lib/errors/errors';
import { mayRollBack, type EndState } from './live';
import { codeSentence } from './log';
import { liveMessages, type LiveKey } from './messages';

/** The result's heading for each end state. */
export const resultHeads: Readonly<Record<EndState, LiveKey>> = {
  created: 'result_checked',
  checked: 'result_checked',
  running: 'titleRun',
  done: 'result_done',
  failed: 'result_failed',
  rolled_back: 'result_rolled_back',
  interrupted: 'result_interrupted',
};

/** One or two sentences on what the end means for the new panel. */
export function useMeaning(state: EndState, report: Report | undefined): string[] {
  const t = useT(liveMessages);
  const te = useT(errorMessages);
  const c = report?.counts;
  switch (state) {
    case 'done':
      return [t('doneBody', { count: c?.moved ?? 0 })];
    case 'failed': {
      const why = report?.error ? codeSentence(report.error) : null;
      const reason = why ? (why.from === 'live' ? t(why.key, why.params) : te(why.key)) : '';
      const stays = mayRollBack(state, c, report);
      return [reason, t(stays ? 'failedStays' : 'failedClean')].filter(Boolean);
    }
    case 'rolled_back':
      return [c?.undone ? t('rolledBackUndo', { count: c.undone }) : t('rolledBackPanel')];
    case 'interrupted':
      return [t('interruptedBody')];
    default:
      return [t('checkedBody')];
  }
}

/** How the reload went and its sentence; null when there was none. */
export function useReload(
  report: Report,
): { kind: Exclude<ReloadOutcome['kind'], 'none'>; text: string } | null {
  const t = useT(liveMessages);
  const r = reloadOutcome(report.reload, report.destination.kind);
  switch (r.kind) {
    case 'none':
      return null;
    case 'manual':
      return { kind: r.kind, text: t('reloadManual', { command: r.command }) };
    case 'error':
      return { kind: r.kind, text: t('reloadError', { command: r.command }) };
    case 'partial':
      return {
        kind: r.kind,
        text: t('reloadPartial', { ok: r.ok, failed: r.failed, command: r.command }),
      };
    default:
      return { kind: r.kind, text: t('reloadOk', { count: r.ok }) };
  }
}
