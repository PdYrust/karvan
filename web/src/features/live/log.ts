import type { Params } from '../../i18n/core';
import type { MigrationEvent } from '../../lib/api/events';
import { errorKey, type ErrorKey } from '../../lib/errors/errors';
import type { LiveKey } from './messages';

/** A sentence from the live view's words, or from the error sentences every page shares (D4). */
export type Sentence =
  { from: 'live'; key: LiveKey; params?: Params } | { from: 'error'; key: ErrorKey };

const stageSentences: Readonly<Record<string, Readonly<Record<string, LiveKey>>>> = {
  check: { start: 'start_check', done: 'done_check', failed: 'failed_check' },
  backup: { start: 'start_backup', done: 'done_backup', failed: 'failed_backup' },
  execute: { start: 'start_execute', done: 'done_execute', failed: 'failed_execute' },
  verify: { start: 'start_verify', done: 'done_verify', failed: 'failed_verify' },
  reload: { start: 'start_reload', done: 'done_reload', failed: 'failed_reload' },
  undo: { start: 'start_undo', done: 'done_undo', failed: 'failed_undo' },
  restore: { start: 'start_restore', done: 'done_restore', failed: 'failed_restore' },
};

const endSentences: Readonly<Record<string, LiveKey>> = {
  done: 'end_done',
  failed: 'end_failed',
  rolled_back: 'end_rolled_back',
  checked: 'end_checked',
};

/** Codes only a run's stream logs (api-migration.md section 3); the rest are shared error codes. */
const runCodes: Readonly<Record<string, LiveKey>> = {
  plan_stale: 'code_plan_stale',
  verify_failed: 'code_verify_failed',
  backup_tool_missing: 'code_backup_tool_missing',
  backup_no_space: 'code_backup_no_space',
  run_failed: 'code_run_failed',
  links_unavailable: 'code_links_unavailable',
  undo_incomplete: 'code_undo_incomplete',
  restore_failed_undoing: 'code_restore_failed_undoing',
  restore_no_backup: 'code_restore_no_backup',
  restore_unavailable: 'code_restore_unavailable',
  ledger_not_updated: 'code_ledger_not_updated',
  report_not_stored: 'code_report_not_stored',
  panel_stopped: 'code_panel_stopped',
  backup_put_back: 'code_backup_put_back',
  panel_started: 'code_panel_started',
  restore_failed: 'code_restore_failed',
};

const own = <T>(map: Readonly<Record<string, T>>, key: string): T | undefined =>
  Object.hasOwn(map, key) ? map[key] : undefined;

/** The sentence for a code a run logged or failed with, with its extras. */
export function codeSentence(
  code: string,
  detail: Readonly<Record<string, unknown>> = {},
): Sentence {
  const key = own(runCodes, code);
  if (key === 'code_undo_incomplete') {
    return { from: 'live', key, params: { count: Number(detail.left) || 0 } };
  }
  if (key) return { from: 'live', key };
  return { from: 'error', key: errorKey(code, detail) };
}

/**
 * A manual reload is the operator's to do, and a reload that missed users
 * did not reload the panel, so neither says it did. Undefined leaves the
 * stage's usual sentence (also for histories from before the stage named
 * its method).
 */
function reloadSentence(state: string, data: MigrationEvent['data']): Sentence | null | undefined {
  if (data.method === 'manual' && state === 'start') return null;
  if (data.method === 'manual' && state === 'done')
    return { from: 'live', key: 'done_reload_manual' };
  if (state === 'done' && data.complete === false)
    return { from: 'live', key: 'done_reload_partial' };
  return undefined;
}

/** One plain sentence for an event of the log; null for events the log leaves out. */
export function eventSentence(e: MigrationEvent): Sentence | null {
  const state = typeof e.data.state === 'string' ? e.data.state : '';
  if (e.type === 'stage') {
    const reload = e.stage === 'reload' ? reloadSentence(state, e.data) : undefined;
    if (reload !== undefined) return reload;
    const key = own(stageSentences, e.stage);
    const sentence = key && own(key, state);
    return sentence ? { from: 'live', key: sentence } : null;
  }
  if (e.type === 'log') {
    return typeof e.data.code === 'string' ? codeSentence(e.data.code, e.data) : null;
  }
  if (e.type === 'done') {
    const key = own(endSentences, state);
    return key ? { from: 'live', key } : null;
  }
  return null;
}
