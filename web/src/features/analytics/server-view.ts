import type { HostMinute, Sample, ServerStats } from '../../lib/api/analytics';
import type { Value } from '../../lib/charts/series';

/** How much live history the page keeps: the 15 minutes the server holds. */
export const liveWindow = 15 * 60_000;

/** A saved minute is drawn at its end, as a sample is drawn at the end of its 5 seconds. */
const minuteEnd = (m: HostMinute) => (m.t + 60) * 1000;

/**
 * Adds a poll's new samples to the ones the page already has. The server
 * sends only samples after the seq it was asked about; when Karvan restarts
 * its seq starts over, and the old samples go. When the samples start over
 * the server also sends the saved minutes before them (backfill), which
 * take the place of what the page had.
 */
export function mergeLive(prev: ServerStats | undefined, next: ServerStats): ServerStats {
  const fresh = next.samples ?? [];
  let samples: Sample[] = fresh;
  let backfill = next.backfill ?? [];
  if (!next.backfill && prev?.samples && next.seq >= prev.seq) {
    const last = prev.samples.at(-1)?.seq ?? 0;
    samples = [...prev.samples, ...fresh.filter((s) => s.seq > last)];
    backfill = prev.backfill ?? [];
  }
  const end = Date.parse(samples.at(-1)?.at ?? '');
  const kept = (t: number) => !Number.isFinite(end) || end - t <= liveWindow;
  return {
    ...next,
    samples: samples.filter((s) => end - Date.parse(s.at) <= liveWindow),
    backfill: backfill.filter((m) => kept(minuteEnd(m))),
  };
}

/** Per-core lines up to this many cores; past it, only the busiest core. */
export const maxCoreLines = 4;

/** One live CPU line: every core together, one core, or the busiest core. */
export interface CpuLine {
  line: 'all' | 'core' | 'busiest';
  /** Which core, from 0, for a core's line. */
  core?: number;
  values: readonly Value[];
}
/**
 * The live CPU lines, every core together first. Saved minutes keep only
 * that line, so it is the one that spans the whole window; the cores (or
 * the busiest core) start with the samples. The overview card, its detail
 * and the analytics chart all take their CPU lines from here.
 */
export function cpuLines(v: ServerView): CpuLine[] {
  return [
    { line: 'all', values: v.cpu },
    ...v.cores.map((values, core) => ({ line: 'core' as const, core, values })),
    ...(v.busiest.length > 0 ? [{ line: 'busiest' as const, values: v.busiest }] : []),
  ];
}
/** Everything the server charts draw, as series over shared times. */
export interface ServerView {
  /** Sample times, ms. */
  times: number[];
  /** Time between samples, ms. */
  step: number;
  /**
   * Live with saved minutes before the samples: the time each point is due
   * after the one before, ms (a minute, then a step; 0 where they do not meet).
   */
  spacing?: number[];
  domain: [number, number];
  live: boolean;
  cpu: Value[];
  /** Range only: the busiest moment of each bucket. */
  cpuPeak: Value[];
  /** Live only: each core, when there are few. */
  cores: Value[][];
  /** Live only: the busiest core of each sample, when there are many. */
  busiest: Value[];
  load1: Value[];
  load5: Value[];
  load15: Value[];
  memUsed: Value[];
  memCache: Value[];
  swap: Value[];
  netRx: Value[];
  netTx: Value[];
  diskRead: Value[];
  diskWrite: Value[];
  tcp: Value[];
  /** Live only: sockets in time-wait. */
  closing: Value[];
  /** Seconds each point stands for, to turn average rates into bytes moved. */
  covered: number[];
}

type Series = Omit<ServerView, 'domain' | 'step' | 'times' | 'spacing'>;

function fromSamples(samples: readonly Sample[], stepSeconds: number): Series {
  const pick = <T>(f: (s: Sample) => T) => samples.map(f);
  const coreCount = Math.max(0, ...samples.map((s) => s.cores.length));
  return {
    live: true,
    covered: pick(() => stepSeconds),
    cpu: pick((s) => s.cpu),
    cpuPeak: [],
    cores:
      coreCount <= maxCoreLines
        ? Array.from({ length: coreCount }, (_, i) => pick((s) => s.cores[i]))
        : [],
    busiest: coreCount > maxCoreLines ? pick((s) => Math.max(0, ...s.cores)) : [],
    load1: pick((s) => s.load[0]),
    load5: pick((s) => s.load[1]),
    load15: pick((s) => s.load[2]),
    memUsed: pick((s) => s.mem.used),
    memCache: pick((s) => s.mem.cache),
    swap: pick((s) => s.mem.swapUsed),
    netRx: pick((s) => s.netRx),
    netTx: pick((s) => s.netTx),
    diskRead: pick((s) => s.diskRead),
    diskWrite: pick((s) => s.diskWrite),
    tcp: pick((s) => s.tcp),
    closing: pick((s) => s.tcpTimeWait),
  };
}

function fromPoints(points: readonly HostMinute[]): Series {
  const pick = <T>(f: (p: HostMinute) => T) => points.map(f);
  return {
    live: false,
    covered: pick((p) => p.minutes * 60),
    cpu: pick((p) => p.cpu),
    cpuPeak: pick((p) => p.cpuMax),
    cores: [],
    busiest: [],
    load1: pick((p) => p.load1),
    load5: [],
    load15: [],
    memUsed: pick((p) => p.memUsed),
    memCache: pick((p) => p.memCache),
    swap: pick((p) => p.swapUsed),
    netRx: pick((p) => p.netRx),
    netTx: pick((p) => p.netTx),
    diskRead: pick((p) => p.diskRead),
    diskWrite: pick((p) => p.diskWrite),
    tcp: pick((p) => p.tcp),
    closing: [],
  };
}

/**
 * Turns the server's answer into chart series. The x axis ends at the
 * server's latest sample rather than this browser's clock, which may be off.
 */
export function serverView(stats: ServerStats, span: number): ServerView {
  const points = stats.points ?? [];
  const live = stats.range === 'live';
  const drawn = live
    ? fromLive(stats.backfill ?? [], stats.samples ?? [], stats.step)
    : { times: points.map((p) => p.t * 1000), series: fromPoints(points) };
  const latest = stats.now ? Date.parse(stats.now.at) : Number.NaN;
  const end = Number.isFinite(latest) ? latest : (drawn.times.at(-1) ?? Date.now());
  return {
    times: drawn.times,
    step: stats.step * 1000,
    domain: [end - span, end],
    ...(drawn.spacing ? { spacing: drawn.spacing } : {}),
    ...drawn.series,
  };
}

/**
 * The live samples, after the saved minutes that come before them. A
 * minute has only what a minute keeps (no cores, no 5 and 15 minute load,
 * no time-wait sockets), so those lines start with the samples. Minutes
 * that reach into the samples are left out, so no time is drawn twice.
 */
function fromLive(
  backfill: readonly HostMinute[],
  samples: readonly Sample[],
  stepSeconds: number,
): { times: number[]; spacing?: number[]; series: Series } {
  const sampleTimes = samples.map((s) => Date.parse(s.at));
  const first = sampleTimes[0] ?? Infinity;
  const minutes = backfill.filter((m) => minuteEnd(m) <= first);
  const series = fromSamples(samples, stepSeconds);
  if (minutes.length === 0) return { times: sampleTimes, series };
  const lead = <T>(f: (m: HostMinute) => T, rest: readonly T[]) => [...minutes.map(f), ...rest];
  const none = (rest: readonly Value[]): Value[] =>
    rest.length > 0 ? [...minutes.map(() => null), ...rest] : [];
  // The samples join the minutes only when the last minute ends in the minute they start in.
  const lastEnd = minuteEnd(minutes.at(-1) as HostMinute);
  const meets = Math.floor(first / 60_000) * 60_000 === lastEnd;
  return {
    times: [...minutes.map(minuteEnd), ...sampleTimes],
    spacing: [
      ...minutes.map(() => 60_000),
      ...sampleTimes.map((_, i) => (i > 0 ? stepSeconds * 1000 : meets ? 60_000 : 0)),
    ],
    series: {
      ...series,
      covered: lead((m) => m.minutes * 60, series.covered),
      cpu: lead((m) => m.cpu, series.cpu),
      cores: series.cores.map(none),
      busiest: none(series.busiest),
      load1: lead((m) => m.load1, series.load1),
      load5: none(series.load5),
      load15: none(series.load15),
      memUsed: lead((m) => m.memUsed, series.memUsed),
      memCache: lead((m) => m.memCache, series.memCache),
      swap: lead((m) => m.swapUsed, series.swap),
      netRx: lead((m) => m.netRx, series.netRx),
      netTx: lead((m) => m.netTx, series.netTx),
      diskRead: lead((m) => m.diskRead, series.diskRead),
      diskWrite: lead((m) => m.diskWrite, series.diskWrite),
      tcp: lead((m) => m.tcp, series.tcp),
      closing: none(series.closing),
    },
  };
}
