import { linePath, lonePoints, runs, splitEdges } from '../../lib/charts/series';
import { chartColor, gridColor } from './palette';
import type { ChartSeries } from './TimeSeries';

/** Where a chart draws: its size, the plot area inside it and the two scales. */
export interface Frame {
  width: number;
  /** Left edge of the plot, after the axis labels. */
  start: number;
  /** Right edge of the plot. */
  end: number;
  top: number;
  bottom: number;
  x: (t: number) => number;
  y: (v: number) => number;
}

function Bars({
  frame,
  times,
  series,
  step,
  active,
}: {
  frame: Frame;
  times: readonly number[];
  series: readonly ChartSeries[];
  step: number;
  active: number | null;
}) {
  const { x, y } = frame;
  const width = Math.max(1, (x(step) - x(0)) * 0.7);
  return times.map((tm, i) => {
    let acc = 0;
    return series.map((s) => {
      const y0 = y(acc);
      acc += s.values[i] ?? 0;
      return (
        <rect
          key={`${s.name}-${String(tm)}`}
          x={x(tm + step / 2) - width / 2}
          y={y(acc)}
          width={width}
          height={Math.max(0, y0 - y(acc))}
          fill={chartColor(s.color)}
          opacity={active === null || active === i ? 1 : 0.55}
        />
      );
    });
  });
}

function Lines({
  frame,
  times,
  series,
  step,
}: {
  frame: Frame;
  times: readonly number[];
  series: readonly ChartSeries[];
  step: number;
}) {
  const { x, y } = frame;
  return series.map((s) => {
    const parts = runs(times, s.values, step * 1.9);
    // A second reading (the peak) is dashed all along; others only at their unfinished edges.
    const { solid, dashed } = s.dashed ? { solid: [], dashed: parts } : splitEdges(parts);
    const stroke = {
      fill: 'none',
      stroke: chartColor(s.color),
      strokeWidth: 1.5,
      strokeLinejoin: 'round',
    } as const;
    return (
      <g key={s.name}>
        <path d={linePath(solid, x, y)} {...stroke} />
        <path d={linePath(dashed, x, y)} {...stroke} strokeDasharray="4 3" />
        {lonePoints(parts).map(([pt, pv]) => (
          <circle key={pt} cx={x(pt)} cy={y(pv)} r={2} fill={chartColor(s.color)} />
        ))}
      </g>
    );
  });
}
/** The drawing itself: grid, series and the crosshair at `active`. */
export function ChartPlot({
  frame,
  height,
  yTicks,
  times,
  series,
  step,
  bars,
  active,
  crosshair,
}: {
  frame: Frame;
  height: number;
  yTicks: readonly number[];
  times: readonly number[];
  series: readonly ChartSeries[];
  step: number;
  bars: boolean;
  active: number | null;
  /** The crosshair's x, in the chart's pixels. */
  crosshair: number;
}) {
  const { y } = frame;
  const box = { width: frame.width, height, 'aria-hidden': true } as const;
  return (
    <>
      <svg {...box} className="absolute inset-0 overflow-visible">
        {yTicks.map((v) => (
          <line
            key={v}
            x1={frame.start}
            x2={frame.end}
            y1={y(v)}
            y2={y(v)}
            stroke={gridColor}
            strokeDasharray={v === 0 ? undefined : '3 3'}
          />
        ))}
        {bars ? (
          // One group grows, so a bar a poll adds just appears with the rest.
          <g className="k-grow-up">
            <Bars frame={frame} times={times} series={series} step={step} active={active} />
          </g>
        ) : null}
      </svg>
      {bars ? null : (
        // The lines are revealed from the start of time to its end when the chart first shows.
        // The reveal clips the outer svg, a box every browser clips (not an inner group).
        <svg {...box} className="k-sweep absolute inset-0 overflow-visible">
          <Lines frame={frame} times={times} series={series} step={step} />
        </svg>
      )}
      <svg {...box} className="absolute inset-0 overflow-visible">
        {active === null ? null : (
          <g>
            <line
              x1={crosshair}
              x2={crosshair}
              y1={frame.top}
              y2={frame.bottom}
              stroke="currentColor"
              className="text-kumo-subtle"
            />
            {bars
              ? null
              : series.map((s) => {
                  const v = s.values[active];
                  return v === null || v === undefined ? null : (
                    <circle
                      key={s.name}
                      cx={crosshair}
                      cy={y(v)}
                      r={3.5}
                      fill={chartColor(s.color)}
                      stroke="var(--color-kumo-base)"
                      strokeWidth={1.5}
                    />
                  );
                })}
          </g>
        )}
      </svg>
    </>
  );
}
