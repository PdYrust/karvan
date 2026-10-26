import { areaPath, linePath, lonePoints, runs } from '../../lib/charts/series';
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
  return series.map((s, i) => {
    const parts = runs(times, s.values, step * 1.9);
    return (
      <g key={s.name}>
        {i === 0 ? (
          <path
            d={areaPath(parts, x, y, y(0))}
            fill={chartColor(s.color)}
            fillOpacity={0.1}
            className="k-grow-up"
          />
        ) : null}
        {/* A dashed line keeps its own dashes, so it fades in instead of drawing. */}
        <path
          d={linePath(parts, x, y)}
          fill="none"
          stroke={chartColor(s.color)}
          strokeWidth={1.5}
          strokeLinejoin="round"
          strokeDasharray={s.dashed ? '4 3' : undefined}
          pathLength={s.dashed ? undefined : 1}
          className={s.dashed ? 'k-draw-fade' : 'k-draw'}
        />
        <g className="k-draw-fade">
          {lonePoints(parts).map(([pt, pv]) => (
            <circle key={pt} cx={x(pt)} cy={y(pv)} r={2} fill={chartColor(s.color)} />
          ))}
        </g>
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
  return (
    <svg
      width={frame.width}
      height={height}
      aria-hidden="true"
      className="absolute inset-0 overflow-visible"
    >
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
      ) : (
        <Lines frame={frame} times={times} series={series} step={step} />
      )}
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
  );
}
