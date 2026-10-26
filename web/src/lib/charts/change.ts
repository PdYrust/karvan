import type { Value } from './series';
/** How a value moved over what Karvan holds: the change, and whether that is good for the operator. */
export interface Change {
  /** A share of the first value (0.12 is 12 %), or, for a share itself, the points it moved (0.02). */
  by: number;
  up: boolean;
  good: boolean;
}
const present = (v: Value): v is number => v !== null && v !== undefined && Number.isFinite(v);
/**
 * From the first value Karvan holds to the latest one. A share (CPU, a
 * disk) changes by points; anything else by a share of where it started.
 * Undefined with fewer than two values, or nothing to compare against.
 */
export function changeOf(
  values: readonly Value[],
  { points = false, risingIsBad = false }: { points?: boolean; risingIsBad?: boolean } = {},
): Change | undefined {
  const known = values.filter(present);
  const first = known[0];
  const last = known.at(-1);
  if (known.length < 2 || first === undefined || last === undefined) return undefined;
  if (!points && first === 0) return undefined;
  const by = points ? last - first : (last - first) / first;
  if (by === 0) return undefined;
  const up = by > 0;
  return { by: Math.abs(by), up, good: up !== risingIsBad };
}
/** The latest value a series holds, or undefined. */
export function latestOf(values: readonly Value[]): number | undefined {
  return values.filter(present).at(-1);
}
