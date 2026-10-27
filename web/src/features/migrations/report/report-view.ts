import type { JobState } from '../../../lib/api/migrations';
import type { Report, ReportUser } from '../../../lib/api/report';
import { mayRollBack } from '../../live/live';

/** The filter buttons over a report's users, in their order. */
export const userFilters = ['all', 'moved', 'left', 'renamed', 'config', 'link'] as const;
export type UserFilter = (typeof userFilters)[number];

const tests: Readonly<Record<UserFilter, (u: ReportUser) => boolean>> = {
  all: () => true,
  moved: (u) => u.moved,
  left: (u) => !u.moved,
  renamed: (u) => u.username !== u.sourceUsername,
  config: (u) => u.changesConfig,
  link: (u) => u.changesLink,
};

export function userMatches(u: ReportUser, f: UserFilter, search: string): boolean {
  const q = search.trim().toLocaleLowerCase();
  if (!tests[f](u)) return false;
  return !q || [u.sourceUsername, u.username].some((n) => n.toLocaleLowerCase().includes(q));
}

/** How many users each filter keeps, for the counts on its button. */
export function filterCounts(users: readonly ReportUser[]): Record<UserFilter, number> {
  const out = Object.fromEntries(userFilters.map((f) => [f, 0])) as Record<UserFilter, number>;
  for (const u of users) {
    for (const f of userFilters) if (tests[f](u)) out[f] += 1;
  }
  return out;
}

/** The things a rollback cannot take back that Karvan has words for, in the report's order. */
export const undoCodes = [
  'notifications_sent',
  'client_connections',
  'core_may_have_loaded',
  'inbound_tag_rows',
  'after_reload',
  'restore_lost_panel_writes',
] as const;
export type UndoCode = (typeof undoCodes)[number];

export function knownUndoCodes(r: Pick<Report, 'notUndoable'>): UndoCode[] {
  const known = new Set<string>(undoCodes);
  return [...new Set(r.notUndoable ?? [])].filter((c): c is UndoCode => known.has(c));
}

/**
 * Until when a rollback can still take the users back: a failed or
 * interrupted job that left users can roll back now; once the panel has
 * reloaded, the undo no longer runs (ADR 0006); otherwise nothing is left
 * to take back.
 */
export type UndoWindow = { kind: 'open' } | { kind: 'reloaded'; at: string } | { kind: 'closed' };

export function undoWindow(state: JobState, report: Report): UndoWindow {
  if (mayRollBack(state, report.counts, report)) return { kind: 'open' };
  if (report.notUndoable?.includes('after_reload')) {
    const reload = report.stages?.find((s) => s.stage === 'reload');
    return { kind: 'reloaded', at: reload?.ended ?? report.endedAt };
  }
  return { kind: 'closed' };
}

/** One user's outcome: moved, moved and then taken back by a rollback, or not moved. */
export function userOutcome(
  u: Pick<ReportUser, 'moved'>,
  state: JobState,
): 'outcomeMoved' | 'outcomeUndone' | 'outcomeLeft' {
  if (!u.moved) return 'outcomeLeft';
  return state === 'rolled_back' ? 'outcomeUndone' : 'outcomeMoved';
}
