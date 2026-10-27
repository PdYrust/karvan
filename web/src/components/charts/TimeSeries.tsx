import { useState, type FocusEvent, type KeyboardEvent, type PointerEvent } from 'react';
import { useT } from '../../i18n/locale';
import { maxValue, nearestIndex, type Value } from '../../lib/charts/series';
import { axisTicks, type AxisKind } from '../../lib/charts/ticks';
import { isDateStep, timeTicks } from '../../lib/charts/time';
import type { Format } from '../../lib/format/format';
import { useFormat } from '../../lib/preferences/region';
import { ChartPlot, type Frame } from './ChartPlot';
import { ChartTable } from './ChartTable';
import { ChartTooltip } from './ChartTooltip';
import { chartMessages } from './messages';
import type { SeriesColor } from './palette';
import { useWidth } from './use-width';

export interface ChartSeries {
  name: string;
  color: SeriesColor;
  values: readonly Value[];
  /** A second reading of the same thing, such as the peak beside the average. */
  dashed?: boolean;
}

export interface TimeSeriesProps {
  /** What the chart shows and over when, for screen readers: "CPU, last 24 hours". */
  label: string;
  /** Sample times in ms, oldest first. */
  times: readonly number[];
  series: readonly ChartSeries[];
  /** The time the x axis covers, in ms. */
  domain: readonly [number, number];
  /** Time between samples in ms; a longer gap leaves a hole. */
  step: number;
  /** Time each point is due after the one before, in ms, where the points are not one step apart. */
  spacing?: readonly number[];
  kind: AxisKind;
  /** A value as the tooltip and the table write it. */
  format: (v: number) => string;
  /** Shown in place of the chart when no series has a value. */
  empty: string;
  /** One bar per sample with the series stacked, instead of lines. */
  bars?: boolean;
  height?: number;
}

const padTop = 8;
const padBottom = 24;
const padEnd = 8;
const labelClass = 'absolute text-sm whitespace-nowrap text-kumo-subtle tabular';

function axisLabel(kind: AxisKind, v: number, fmt: Format): string {
  switch (kind) {
    case 'percent':
      return fmt.percent(v, 0);
    case 'bytes':
      return fmt.bytes(v);
    case 'rate':
      return fmt.rate(v);
    case 'count':
      return fmt.compact(v);
    default:
      return fmt.number(v, 1);
  }
}

function stackedMax(series: readonly ChartSeries[], n: number): number {
  let max = 0;
  for (let i = 0; i < n; i++) {
    max = Math.max(
      max,
      series.reduce((sum, s) => sum + (s.values[i] ?? 0), 0),
    );
  }
  return max;
}

/**
 * A time chart in SVG: thin lines (or stacked bars), a faint area under
 * the first series, dashed grid lines and round ticks. A pointer or the
 * arrow keys move a crosshair that reads every series. Time runs left to
 * right in both languages.
 */
export function TimeSeries(props: TimeSeriesProps) {
  const { label, times, series, domain, step, kind, format, empty } = props;
  const { bars = false, height = 176, spacing } = props;
  const fmt = useFormat();
  const t = useT(chartMessages);
  const [ref, width] = useWidth<HTMLDivElement>();
  const [active, setActive] = useState<number | null>(null);

  const hasData = series.some((s) => s.values.some((v) => v !== null && v !== undefined));
  const peak = bars ? stackedMax(series, times.length) : maxValue(series.map((s) => s.values));
  const yTicks = axisTicks(kind, peak);
  const yMax = yTicks.at(-1) || 1;
  const yLabels = yTicks.map((v) => axisLabel(kind, v, fmt));
  const start = Math.ceil(Math.max(...yLabels.map((l) => l.length)) * 7.5) + 14;
  const [t0, t1] = domain;
  const plotW = Math.max(width - start - padEnd, 1);
  const bottom = height - padBottom;
  const frame: Frame = {
    width,
    start,
    end: width - padEnd,
    top: padTop,
    bottom,
    x: (tm) => start + ((tm - t0) / (t1 - t0 || 1)) * plotW,
    y: (v) => bottom - (Math.min(v, yMax) / yMax) * (bottom - padTop),
  };
  const center = bars ? step / 2 : 0;
  const xTicks = timeTicks(t0, t1, fmt.timeZone, width < 480 ? 3 : 5);
  const tickTime = (tm: number) =>
    isDateStep(xTicks.step) ? fmt.shortDate(new Date(tm)) : fmt.time(new Date(tm));
  const valueText = (v: Value) => (v === null || v === undefined ? t('noValue') : format(v));
  const last = times.length - 1;
  const current = Math.max(0, active ?? last);
  const describe = (i: number) =>
    `${fmt.dateTime(new Date(times[i] ?? 0))}: ${series
      .map((s) => `${s.name} ${valueText(s.values[i])}`)
      .join(fmt.locale === 'fa' ? '، ' : ', ')}`;
  const crosshair = active === null ? 0 : frame.x((times[active] ?? 0) + center);

  function pointAt(e: PointerEvent<HTMLDivElement>) {
    const px = e.clientX - e.currentTarget.getBoundingClientRect().left;
    setActive(nearestIndex(times, t0 + ((px - start) / plotW) * (t1 - t0) - center));
  }

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    const jump = Math.max(1, Math.round(times.length / 10));
    const moves: Record<string, number> = {
      ArrowRight: current + 1,
      ArrowUp: current + 1,
      ArrowLeft: current - 1,
      ArrowDown: current - 1,
      PageUp: current + jump,
      PageDown: current - jump,
      Home: 0,
      End: last,
    };
    const next = moves[e.key];
    if (next === undefined) return;
    e.preventDefault();
    setActive(Math.min(last, Math.max(0, next)));
  }

  function onFocus(e: FocusEvent<HTMLDivElement>) {
    // A click focuses too; only a keyboard focus starts at the latest point.
    if (e.currentTarget.matches(':focus-visible')) setActive(last);
  }

  return (
    <div ref={ref} className="relative" style={{ height }}>
      {!hasData ? (
        <p className="grid h-full place-items-center text-sm text-kumo-subtle">{empty}</p>
      ) : width > 0 ? (
        <>
          <ChartPlot
            frame={frame}
            height={height}
            yTicks={yTicks}
            times={times}
            series={series}
            step={step}
            spacing={spacing}
            bars={bars}
            active={active}
            crosshair={crosshair}
          />
          {/* Labels are HTML, so Persian digits and words read in the page direction. */}
          {yTicks.map((v, i) => (
            <span
              key={v}
              aria-hidden="true"
              className={labelClass}
              // The chart runs left to right in both languages: physical on purpose.
              style={{ left: 0, width: start - 8, top: frame.y(v) - 11, textAlign: 'right' }}
            >
              {yLabels[i]}
            </span>
          ))}
          {xTicks.ticks.map((tm) => {
            const px = frame.x(tm);
            if (px < start + 16 || px > width - 16) return null;
            return (
              <span
                key={tm}
                aria-hidden="true"
                className={labelClass}
                style={{ left: px, top: bottom + 3, transform: 'translateX(-50%)' }}
              >
                {tickTime(tm)}
              </span>
            );
          })}
          <div
            role="slider"
            tabIndex={0}
            aria-label={label}
            aria-orientation="horizontal"
            aria-valuemin={0}
            aria-valuemax={Math.max(0, last)}
            aria-valuenow={current}
            aria-valuetext={describe(current)}
            className="absolute inset-0 touch-pan-y rounded-md"
            onPointerMove={pointAt}
            onPointerDown={pointAt}
            onPointerLeave={() => setActive(null)}
            onFocus={onFocus}
            onBlur={() => setActive(null)}
            onKeyDown={onKeyDown}
          />
          {active === null ? null : (
            <ChartTooltip
              x={crosshair}
              flip={crosshair > width / 2}
              title={fmt.dateTime(new Date(times[active] ?? 0))}
              series={series}
              index={active}
              valueText={valueText}
            />
          )}
          <ChartTable
            caption={label}
            times={times}
            series={series}
            timeLabel={(tm) => fmt.dateTime(new Date(tm))}
            valueText={valueText}
          />
        </>
      ) : null}
    </div>
  );
}
