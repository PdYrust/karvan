import type { Value } from '../../lib/charts/series';
import { chartColor } from './palette';
import type { ChartSeries } from './TimeSeries';

/**
 * The values at the crosshair. It sits beside the crosshair and moves to
 * the other side past the middle, so it never covers the point it reads.
 */
export function ChartTooltip({
  x,
  flip,
  title,
  series,
  index,
  valueText,
}: {
  /** Crosshair position in the chart's own left-to-right pixels. */
  x: number;
  flip: boolean;
  title: string;
  series: readonly ChartSeries[];
  index: number;
  valueText: (v: Value) => string;
}) {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute top-0 z-10 flex min-w-40 flex-col gap-1 rounded-md bg-kumo-base px-3 py-2 text-sm shadow-md ring ring-kumo-line"
      // The chart runs left to right in both languages, so this is physical on purpose.
      style={{ left: flip ? x - 12 : x + 12, transform: flip ? 'translateX(-100%)' : undefined }}
    >
      <span className="whitespace-nowrap text-kumo-subtle tabular">{title}</span>
      {series.map((s) => (
        <span key={s.name} className="flex items-center gap-2 whitespace-nowrap">
          <span
            className="size-2 shrink-0 rounded-full"
            style={{ background: chartColor(s.color) }}
          />
          <span className="text-kumo-default">{s.name}</span>
          <span className="ms-auto ps-4 font-medium text-kumo-strong tabular">
            {valueText(s.values[index])}
          </span>
        </span>
      ))}
    </div>
  );
}
