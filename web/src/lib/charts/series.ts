/** A value at one time; null where the sample is missing. */
export type Value = number | null | undefined;

export type Point = readonly [t: number, v: number];

/**
 * Splits a series into runs of points that follow each other. A missing
 * value, or a gap of more than maxGap between two times, ends a run, so the
 * chart leaves a hole instead of drawing a line through time it never saw.
 */
export function runs(
  times: readonly number[],
  values: readonly Value[],
  maxGap: number,
): Point[][] {
  const out: Point[][] = [];
  let run: Point[] = [];
  let last = -Infinity;
  times.forEach((t, i) => {
    const v = values[i];
    if (v === null || v === undefined || !Number.isFinite(v)) {
      if (run.length > 0) out.push(run);
      run = [];
      return;
    }
    if (run.length > 0 && t - last > maxGap) {
      out.push(run);
      run = [];
    }
    run.push([t, v]);
    last = t;
  });
  if (run.length > 0) out.push(run);
  return out;
}

type Scale = (n: number) => number;

const fixed = (n: number) => Math.round(n * 10) / 10;

/** An SVG path through every run: "M x y L x y ...", one M per run. */
export function linePath(parts: readonly Point[][], x: Scale, y: Scale): string {
  return parts
    .filter((r) => r.length > 1)
    .map((r) =>
      r.map(([t, v], i) => `${i === 0 ? 'M' : 'L'}${fixed(x(t))} ${fixed(y(v))}`).join(''),
    )
    .join('');
}

/** The area under every run down to the baseline, closed per run. */
export function areaPath(parts: readonly Point[][], x: Scale, y: Scale, base: number): string {
  return parts
    .filter((r) => r.length > 1)
    .map((r) => {
      const first = r[0] as Point;
      const last = r[r.length - 1] as Point;
      const top = r.map(([t, v]) => `L${fixed(x(t))} ${fixed(y(v))}`).join('');
      return `M${fixed(x(first[0]))} ${fixed(base)}${top}L${fixed(x(last[0]))} ${fixed(base)}Z`;
    })
    .join('');
}

/** Points that stand alone between two gaps; a line cannot show them, so they get a dot. */
export function lonePoints(parts: readonly Point[][]): Point[] {
  return parts.filter((r) => r.length === 1).map((r) => r[0] as Point);
}

/** Index of the time nearest to t; times are sorted. -1 when there are none. */
export function nearestIndex(times: readonly number[], t: number): number {
  if (times.length === 0) return -1;
  let lo = 0;
  let hi = times.length - 1;
  while (hi - lo > 1) {
    const mid = (lo + hi) >> 1;
    if ((times[mid] as number) <= t) lo = mid;
    else hi = mid;
  }
  return Math.abs(t - (times[lo] as number)) <= Math.abs((times[hi] as number) - t) ? lo : hi;
}

/** The largest value across series, ignoring missing ones; 0 when there is none. */
export function maxValue(series: readonly (readonly Value[])[]): number {
  let max = 0;
  for (const s of series) {
    for (const v of s) {
      if (v !== null && v !== undefined && v > max) max = v;
    }
  }
  return max;
}

/** Up to n indexes spread evenly from first to last, for a table that sums up a long series. */
export function spreadIndexes(length: number, n: number): number[] {
  if (length <= n) return Array.from({ length }, (_, i) => i);
  if (n <= 1) return [length - 1];
  return Array.from({ length: n }, (_, i) => Math.round((i * (length - 1)) / (n - 1)));
}
/**
 * The first and last segment of a series, drawn dashed: a bucket at either
 * edge of the window is not complete. The rest stays solid.
 */
export function splitEdges(parts: readonly Point[][]): { solid: Point[][]; dashed: Point[][] } {
  const solid = parts.map((p) => [...p]);
  const dashed: Point[][] = [];
  const first = solid[0];
  if (first && first.length >= 2) {
    dashed.push(first.slice(0, 2));
    solid[0] = first.slice(1);
  }
  const at = solid.length - 1;
  const last = solid[at];
  if (last && last.length >= 2) {
    dashed.push(last.slice(-2));
    solid[at] = last.slice(0, -1);
  }
  return { solid: solid.filter((p) => p.length >= 2), dashed };
}
/** Series on the same times added up; a time where every series is missing stays missing. */
export function sumValues(list: readonly (readonly Value[])[]): Value[] {
  const n = Math.max(0, ...list.map((l) => l.length));
  return Array.from({ length: n }, (_, i) => {
    let sum: number | null = null;
    for (const l of list) {
      const v = l[i];
      if (v !== null && v !== undefined && Number.isFinite(v)) sum = (sum ?? 0) + v;
    }
    return sum;
  });
}
