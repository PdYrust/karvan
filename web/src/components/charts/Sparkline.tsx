import { cn } from '@cloudflare/kumo/utils';
import { areaPath, linePath, runs, type Value } from '../../lib/charts/series';
import { chartColor, type SeriesColor } from './palette';

const h = 32;

/**
 * The range a sparkline spans when it has no fixed top: from the lowest to
 * the highest value, so a slow climb (users, size) still shows its shape.
 */
function spread(values: readonly Value[]): [number, number] {
  let lo = Infinity;
  let hi = -Infinity;
  for (const v of values) {
    if (v === null || v === undefined || !Number.isFinite(v)) continue;
    lo = Math.min(lo, v);
    hi = Math.max(hi, v);
  }
  if (!Number.isFinite(lo)) return [0, 1];
  const pad = hi > lo ? (hi - lo) * 0.15 : Math.max(Math.abs(hi), 1);
  return [Math.max(0, lo - pad), hi + pad];
}

/**
 * A small line with a faint area under it, along the bottom of a stat tile.
 * Decoration only: the tile shows the number in text.
 */
export function Sparkline({
  times,
  values,
  domain,
  step,
  max,
  color = 1,
  className,
}: {
  times: readonly number[];
  values: readonly Value[];
  domain: readonly [number, number];
  /** Time between samples, in ms; longer gaps break the line. */
  step: number;
  /** A fixed scale from zero, such as 1 for a share; else the values' own range. */
  max?: number;
  color?: SeriesColor;
  className?: string;
}) {
  const [t0, t1] = domain;
  const [lo, hi] = max === undefined ? spread(values) : [0, max];
  const x = (t: number) => ((t - t0) / (t1 - t0 || 1)) * 100;
  // One unit of room at the top, so a flat line at the maximum stays visible.
  const y = (v: number) => h - ((Math.min(v, hi) - lo) / (hi - lo)) * (h - 2) - 1;
  const parts = runs(times, values, step * 1.9);
  return (
    <svg
      viewBox={`0 0 100 ${String(h)}`}
      preserveAspectRatio="none"
      aria-hidden="true"
      className={cn('block h-10 w-full', className)}
    >
      <path d={areaPath(parts, x, y, h)} fill={chartColor(color)} fillOpacity={0.12} />
      <path
        d={linePath(parts, x, y)}
        fill="none"
        stroke={chartColor(color)}
        strokeWidth={1.5}
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}
