const minute = 60_000;
const hour = 60 * minute;
const day = 24 * hour;

/** Steps a time axis may use, smallest first. */
const steps = [
  minute,
  2 * minute,
  5 * minute,
  10 * minute,
  15 * minute,
  30 * minute,
  hour,
  2 * hour,
  3 * hour,
  6 * hour,
  12 * hour,
  day,
  2 * day,
  7 * day,
];

const clocks = new Map<string, Intl.DateTimeFormat>();

/** How far a zone's clock is ahead of UTC at a moment, in ms (Tehran: 3.5 hours). */
export function zoneOffset(timeZone: string, at: number): number {
  let f = clocks.get(timeZone);
  if (!f) {
    f = new Intl.DateTimeFormat('en-US', {
      timeZone,
      hourCycle: 'h23',
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
      hour: 'numeric',
      minute: 'numeric',
      second: 'numeric',
    });
    clocks.set(timeZone, f);
  }
  const p: Record<string, number> = {};
  for (const part of f.formatToParts(at)) {
    if (part.type !== 'literal') p[part.type] = Number(part.value);
  }
  const local = Date.UTC(
    p.year ?? 1970,
    (p.month ?? 1) - 1,
    p.day ?? 1,
    p.hour ?? 0,
    p.minute ?? 0,
    p.second ?? 0,
  );
  return local - Math.floor(at / 1000) * 1000;
}

export interface TimeTicks {
  ticks: number[];
  /** The step between ticks, in ms; a day or more means the labels are dates. */
  step: number;
}

/**
 * Round moments between from and to (ms), on the zone's own clock: every
 * 5 minutes, every 6 hours, each midnight. At most `count` of them.
 */
export function timeTicks(from: number, to: number, timeZone: string, count = 5): TimeTicks {
  const span = Math.max(to - from, minute);
  const step = steps.find((s) => span / s <= count) ?? (steps.at(-1) as number);
  const offset = zoneOffset(timeZone, from);
  const ticks: number[] = [];
  for (let t = Math.ceil((from + offset) / step) * step - offset; t <= to; t += step) {
    ticks.push(t);
  }
  return { ticks, step };
}

/** True when ticks this far apart should be labelled with dates rather than clock times. */
export function isDateStep(step: number): boolean {
  return step >= day;
}
