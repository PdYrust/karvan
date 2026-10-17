import { zoneOffset } from '../../../lib/charts/time';

export const hour = 3_600_000;
export const day = 24 * hour;

export interface Buckets {
  /** Start of each bucket, ms. */
  times: number[];
  /** One array per series, a sum per bucket; 0 where nothing happened. */
  series: number[][];
  step: number;
}

/**
 * Sums timed values into hours or days from `from` to `to`. Days follow
 * the zone's own midnight; an hour or a day with nothing in it counts 0,
 * since a quiet hour is a real zero, not a missing sample.
 */
export function sumBuckets(
  items: readonly { at: number; values: readonly number[] }[],
  from: number,
  to: number,
  size: number,
  timeZone: string,
  seriesCount: number,
): Buckets {
  const offset = size >= day ? zoneOffset(timeZone, to) : 0;
  const startOf = (t: number) => Math.floor((t + offset) / size) * size - offset;
  const first = startOf(from);
  const n = Math.max(0, Math.floor((startOf(to) - first) / size) + 1);
  const times = Array.from({ length: n }, (_, i) => first + i * size);
  const series = Array.from({ length: seriesCount }, () => new Array<number>(n).fill(0));
  for (const item of items) {
    const i = (startOf(item.at) - first) / size;
    if (i < 0 || i >= n) continue;
    item.values.forEach((v, s) => {
      const row = series[s];
      if (row) row[i] = (row[i] ?? 0) + v;
    });
  }
  return { times, series, step: size };
}

/** Hours for a day or less, days beyond that, so a bar stays wide enough to read. */
export function bucketSize(span: number): number {
  return span <= day ? hour : day;
}
