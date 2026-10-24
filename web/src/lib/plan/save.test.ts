import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/client';
import type { PlanSaved, PlanState } from '../api/migrations';
import { PlanConflict, saveEdit } from './save';
import { setTargets, type Plan } from './plan';

const plan = (targets: string[] | null): Plan => ({
  version: 0,
  now: '2026-10-17T09:06:00Z',
  admin: {},
  targets,
  tree: null,
});
const saved = (planVersion: number): PlanSaved => ({
  planVersion,
  blockers: 0,
  stable: true,
  groups: [],
  conflicts: [],
  impact: { users: 3, moving: 3 },
});
const stale = () => new ApiError(412, 'stale_plan', undefined, { planVersion: 5 });

describe('saveEdit', () => {
  it('sends the edited plan with the version it was made on', async () => {
    const save = vi.fn(() => Promise.resolve(saved(3)));
    const load = vi.fn<() => Promise<PlanState>>();
    const out = await saveEdit(
      setTargets(['3']),
      { planVersion: 2, plan: plan(null) },
      { load, save },
    );
    expect(save).toHaveBeenCalledWith(2, plan(['3']));
    expect(load).not.toHaveBeenCalled();
    expect(out).toEqual({ plan: plan(['3']), saved: saved(3) });
  });

  it('after a 412 reads the plan again and replays the edit on it once', async () => {
    const save = vi
      .fn<(v: number, p: Plan) => Promise<PlanSaved>>()
      .mockRejectedValueOnce(stale())
      .mockResolvedValueOnce(saved(6));
    const other = { ...plan(null), admin: { username: 'shop_2' } };
    const load = vi.fn(() => Promise.resolve({ planVersion: 5, plan: other }));
    const out = await saveEdit(
      setTargets(['3']),
      { planVersion: 2, plan: plan(null) },
      { load, save },
    );
    expect(save).toHaveBeenLastCalledWith(5, { ...other, targets: ['3'] });
    expect(out.saved.planVersion).toBe(6);
  });

  it('asks the operator after a second conflict', async () => {
    const save = vi.fn(() => Promise.reject(stale()));
    const load = vi.fn(() => Promise.resolve({ planVersion: 5, plan: plan(null) }));
    await expect(
      saveEdit(setTargets(['3']), { planVersion: 2, plan: plan(null) }, { load, save }),
    ).rejects.toBeInstanceOf(PlanConflict);
    expect(save).toHaveBeenCalledTimes(2);
  });

  it('passes any other failure on as it is', async () => {
    const busy = new ApiError(409, 'busy');
    const save = vi.fn(() => Promise.reject(busy));
    const load = vi.fn<() => Promise<PlanState>>();
    await expect(
      saveEdit(setTargets(['3']), { planVersion: 2, plan: plan(null) }, { load, save }),
    ).rejects.toBe(busy);
    expect(load).not.toHaveBeenCalled();
  });
});
