import { useT } from '../../i18n/locale';
import { areaPath, linePath, runs, splitEdges, type Value } from '../../lib/charts/series';
import { chartMessages } from './messages';
import { chartColor, gridColor } from './palette';
import { useSize } from './use-width';
import { cn } from '@cloudflare/kumo/utils';

/**
 * The range the line spans when it has no fixed top: from the lowest to the
 * highest value, so a slow climb (users, size) still shows its shape.
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

/** A faint, wide wave with a few soft hills and a "No data" pill, in place of a chart with nothing to draw. */
function NoData() {
  const t = useT(chartMessages);
  return (
    <div className="relative size-full">
      <svg
        viewBox="0 0 300 100"
        preserveAspectRatio="none"
        aria-hidden="true"
        className="absolute inset-0 size-full"
      >
        <path
          d="M0 70 C40 70 50 40 90 40 S140 72 180 72 S230 44 260 46 S290 62 300 60"
          fill="none"
          stroke={gridColor}
          strokeWidth={1.5}
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      <span className="absolute inset-0 grid place-items-center">
        <span className="rounded-full bg-kumo-base px-2.5 py-0.5 text-sm text-kumo-subtle ring ring-kumo-hairline">
          {t('noValue')}
        </span>
      </span>
    </div>
  );
}

/**
 * The chart of a compact card: full bleed, no axes, ticks or grid, only the
 * line and its light area, the first and last segment dashed. Revealed from
 * the start of time when it first shows. Decoration: the card shows the number.
 */
export function CompactChart({
  times,
  values,
  domain,
  step,
  max,
  className,
}: {
  times: readonly number[];
  values: readonly Value[];
  domain: readonly [number, number];
  /** Time between samples, in ms; longer gaps break the line. */
  step: number;
  /** A fixed scale from zero, such as 1 for a share; else the values' own range. */
  max?: number;
  /** Its height, by screen size ("h-18 sm:h-14"). */
  className: string;
}) {
  // Always the same box, so it is measured even when the data arrive after it mounts.
  const [ref, { width, height }] = useSize<HTMLDivElement>();
  const parts = runs(times, values, step * 1.9);
  const empty = !parts.some((p) => p.length > 0);
  const [t0, t1] = domain;
  const [lo, hi] = max === undefined ? spread(values) : [0, max];
  const x = (t: number) => ((t - t0) / (t1 - t0 || 1)) * width;
  // Two pixels of room at the top, so a flat line at the maximum stays visible.
  const y = (v: number) => height - ((Math.min(v, hi) - lo) / (hi - lo || 1)) * (height - 4) - 2;
  const { solid, dashed } = splitEdges(parts);
  const color = chartColor(1);
  const stroke = {
    fill: 'none',
    stroke: color,
    strokeWidth: 1.5,
    strokeLinejoin: 'round',
  } as const;
  return (
    <div ref={ref} className={cn('relative', className)}>
      {empty ? <NoData /> : null}
      {!empty && width > 0 && height > 0 ? (
        // The line and its area are revealed together from the start of time (the svg's box is clipped).
        <svg width={width} height={height} aria-hidden="true" className="k-sweep absolute inset-0">
          <path d={areaPath(parts, x, y, height)} fill={color} fillOpacity={0.12} />
          <path d={linePath(solid, x, y)} {...stroke} />
          <path d={linePath(dashed, x, y)} {...stroke} strokeDasharray="4 3" />
        </svg>
      ) : null}
    </div>
  );
}
