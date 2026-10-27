import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ChartPlot, type Frame } from './ChartPlot';
import type { ChartSeries } from './TimeSeries';

// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement;
let root: Root;

const frame: Frame = {
  width: 400,
  start: 40,
  end: 390,
  top: 10,
  bottom: 150,
  x: (t) => 40 + t,
  y: (v) => 150 - v,
};

function draw(count: number, bars = false, dashed = false) {
  const times = Array.from({ length: count }, (_, i) => i * 10);
  const series: ChartSeries[] = [
    { name: 'cpu', color: 1, values: times.map((_, i) => i + 1), dashed },
  ];
  act(() =>
    root.render(
      <ChartPlot
        frame={frame}
        height={176}
        yTicks={[0, 50]}
        times={times}
        series={series}
        step={10}
        bars={bars}
        active={null}
        crosshair={0}
      />,
    ),
  );
}

beforeEach(() => {
  host = document.createElement('div');
  root = createRoot(host);
});

afterEach(() => {
  act(() => root.unmount());
});

describe('ChartPlot draw-in', () => {
  it('reveals the lines once from the start of time, without an area; a poll keeps the group', () => {
    draw(5);
    const sweep = host.querySelector('svg.k-sweep');
    expect(sweep?.querySelectorAll('path')).toHaveLength(2);
    expect(host.querySelectorAll('path[fill-opacity]')).toHaveLength(0);
    draw(6);
    // The same node: the reveal, run on mount, does not start again.
    expect(host.querySelector('svg.k-sweep')).toBe(sweep);
  });

  it('dashes only the first and last segment of a line', () => {
    draw(5);
    const [solid, edges] = host.querySelectorAll('svg.k-sweep path');
    expect(solid?.hasAttribute('stroke-dasharray')).toBe(false);
    expect(edges?.getAttribute('stroke-dasharray')).toBe('4 3');
    // Two segments, each a move and a line.
    expect(edges?.getAttribute('d')?.match(/M/g)).toHaveLength(2);
  });

  it('dashes a second reading all along', () => {
    draw(5, false, true);
    const [solid, dashed] = host.querySelectorAll('svg.k-sweep path');
    expect(solid?.getAttribute('d')).toBe('');
    expect(dashed?.getAttribute('d')?.match(/M/g)).toHaveLength(1);
  });

  it('grows bars as one group, so a bar a poll adds does not grow on its own', () => {
    draw(5, true);
    const group = host.querySelector('g.k-grow-up');
    expect(group?.querySelectorAll('rect')).toHaveLength(5);
    draw(6, true);
    expect(host.querySelector('g.k-grow-up')).toBe(group);
    expect(group?.querySelectorAll('rect')).toHaveLength(6);
    expect(host.querySelectorAll('rect[class]')).toHaveLength(0);
  });
});
