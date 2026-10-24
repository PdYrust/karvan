import { ApiError } from '../api/client';
import type { PlanSaved, PlanState } from '../api/migrations';
import type { Plan, PlanEdit } from './plan';

/** The plan changed again while an edit was replayed: the operator decides (D7). */
export class PlanConflict extends Error {
  constructor() {
    super('plan_conflict');
    this.name = 'PlanConflict';
  }
}

const isStale = (err: unknown) => err instanceof ApiError && err.status === 412;

/**
 * Saves one edit: the whole plan with the version it was made on (D6). When
 * another tab or a new read changed the plan first (412), it reads the plan
 * again and replays the edit on it once; a second conflict is a
 * PlanConflict (D7).
 */
export async function saveEdit(
  edit: PlanEdit,
  current: PlanState,
  io: {
    load: () => Promise<PlanState>;
    save: (version: number, plan: Plan) => Promise<PlanSaved>;
  },
): Promise<{ plan: Plan; saved: PlanSaved }> {
  const first = edit(current.plan);
  try {
    return { plan: first, saved: await io.save(current.planVersion, first) };
  } catch (err) {
    if (!isStale(err)) throw err;
  }
  const fresh = await io.load();
  const again = edit(fresh.plan);
  try {
    return { plan: again, saved: await io.save(fresh.planVersion, again) };
  } catch (err) {
    throw isStale(err) ? new PlanConflict() : err;
  }
}
