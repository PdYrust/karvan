/** Drops floating point noise such as 0.30000000000000004. */
function clean(n: number): number {
  return Number(n.toPrecision(12));
}

/** The step a round axis uses: 1, 2, 2.5 or 5 times a power of ten, at least `rough`. */
export function niceStep(rough: number): number {
  if (!(rough > 0) || !Number.isFinite(rough)) return 1;
  const power = 10 ** Math.floor(Math.log10(rough));
  const f = clean(rough / power);
  const m = f <= 1 ? 1 : f <= 2 ? 2 : f <= 2.5 ? 2.5 : f <= 5 ? 5 : 10;
  return clean(m * power);
}

/** Round ticks from zero to the first one at or above max: 0, 25, 50, 75, 100. */
export function niceTicks(max: number, count = 4): number[] {
  const top = max > 0 && Number.isFinite(max) ? max : 1;
  const step = niceStep(top / count);
  const n = Math.max(1, Math.ceil(clean(top / step)));
  return Array.from({ length: n + 1 }, (_, i) => clean(i * step));
}

const kib = 1024;

/**
 * Ticks for a byte axis that are round in the unit the top value is shown
 * in, so a memory axis reads 0, 5 GB, 10 GB rather than 4.66 GB steps.
 */
export function byteTicks(max: number, count = 4): number[] {
  let unit = 1;
  while (max / unit >= kib && unit < kib ** 5) unit *= kib;
  return niceTicks(max / unit, count).map((v) => v * unit);
}

/** A share from 0 to 1 always gets the same quarters, so charts compare at a glance. */
export const percentTicks: readonly number[] = [0, 0.25, 0.5, 0.75, 1];

/** What a chart's values are, which decides its ticks and their labels. */
export type AxisKind = 'percent' | 'bytes' | 'rate' | 'count' | 'number';

/** Ticks for an axis whose largest value is max. */
export function axisTicks(kind: AxisKind, max: number): number[] {
  switch (kind) {
    case 'percent':
      return max > 1 ? niceTicks(max) : [...percentTicks];
    case 'bytes':
    case 'rate':
      return byteTicks(max);
    default:
      return niceTicks(max);
  }
}
