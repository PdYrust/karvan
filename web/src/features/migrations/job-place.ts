import { panelSearchNames } from '../../components/panels/PanelLogo';
import { prefillQuery } from '../../lib/api/links';
import type { JobRow, JobState } from '../../lib/api/migrations';
import { newMigrationPath, stepPath } from '../wizard/steps';

/** The state filter a list offers; drafts stand for both draft states. */
export const stateFilters = [
  'all',
  'draft',
  'running',
  'done',
  'failed',
  'rolled_back',
  'interrupted',
] as const;
export type StateFilter = (typeof stateFilters)[number];

/** The states a filter asks the server for; none for all. */
export function statesOf(f: StateFilter): JobState[] {
  if (f === 'all') return [];
  if (f === 'draft') return ['created', 'checked'];
  return [f];
}

/** Jobs that ended with a report: the Reports page lists these. */
export const reportStates: readonly JobState[] = ['done', 'failed', 'rolled_back'];

export function isDraft(state: JobState): boolean {
  return state === 'created' || state === 'checked';
}

/** A draft whose old panel login Karvan let go (2 hours idle or a restart). */
export function loginDropped(job: Pick<JobRow, 'state' | 'loginHeld'>): boolean {
  return isDraft(job.state) && !job.loginHeld;
}

export function reportPath(id: string): string {
  return `/migrations/${encodeURIComponent(id)}/report`;
}

/**
 * Where `/migrations/:id` takes the operator: a read still going to its
 * counter, a draft to its steps, a run to the live view, a finished job
 * to its report. A draft Karvan signed out of can only start again (after
 * a restart its steps answer 404), so it opens Connect from its address.
 */
export function jobPlace(
  job: Pick<JobRow, 'id' | 'state' | 'loginHeld' | 'panel' | 'source'>,
): string {
  if (loginDropped(job) && job.source.origin) return againPath(job);
  switch (job.state) {
    case 'created':
    case 'checked':
      return job.loginHeld && job.state === 'created'
        ? `${newMigrationPath}?job=${encodeURIComponent(job.id)}`
        : stepPath('summary', job.id);
    case 'done':
    case 'rolled_back':
      return reportPath(job.id);
    default:
      return `/migrations/${encodeURIComponent(job.id)}/live`;
  }
}

/** Connect again with the same destination, address and kind; the sign-in is typed again. */
export function againPath(job: Pick<JobRow, 'panel' | 'source'>): string {
  const query = prefillQuery({
    panelId: job.panel.id,
    sourceKind: job.source.kind,
    sourceUrl: job.source.origin,
  });
  return query ? `${newMigrationPath}?${query}` : newMigrationPath;
}

/** Whether a job matches the search field: its admin, the old panel's address, a panel name or kind. */
export function matchesJob(job: JobRow, query: string): boolean {
  const q = query.trim().toLocaleLowerCase();
  if (!q) return true;
  return [
    job.source.admin,
    job.source.origin,
    job.panel.name,
    panelSearchNames[job.source.kind],
    job.panel.kind ? panelSearchNames[job.panel.kind] : undefined,
  ].some((v) => v?.toLocaleLowerCase().includes(q));
}
