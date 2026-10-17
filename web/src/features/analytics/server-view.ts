import type { HostMinute, Sample, ServerStats } from '../../lib/api/analytics';
import type { Value } from '../../lib/charts/series';

/** How much live history the page keeps: the 15 minutes the server holds. */
export const liveWindow = 15 * 60_000;

/**
 * Adds a poll's new samples to the ones the page already has. The server
 * sends only samples after the seq it was asked about; when Karvan restarts
 * its seq starts over, and the old samples go.
 */
export function mergeLive(prev: ServerStats | undefined, next: ServerStats): ServerStats {
  const fresh = next.samples ?? [];
  let samples: Sample[] = fresh;
  if (prev?.samples && next.seq >= prev.seq) {
    const last = prev.samples.at(-1)?.seq ?? 0;
    samples = [...prev.samples, ...fresh.filter((s) => s.seq > last)];
  }
  const end = Date.parse(samples.at(-1)?.at ?? '');
  return { ...next, samples: samples.filter((s) => end - Date.parse(s.at) <= liveWindow) };
}

/** Per-core lines up to this many cores; past it, only the busiest core. */
export const maxCoreLines = 4;

/** Everything the server charts draw, as series over shared times. */
export interface ServerView {
  /** Sample times, ms. */
  times: number[];
  /** Time between samples, ms. */
  step: number;
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

type Series = Omit<ServerView, 'domain' | 'step' | 'times'>;

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
  const samples = stats.samples ?? [];
  const points = stats.points ?? [];
  const live = stats.range === 'live';
  const times = live ? samples.map((s) => Date.parse(s.at)) : points.map((p) => p.t * 1000);
  const latest = stats.now ? Date.parse(stats.now.at) : Number.NaN;
  const end = Number.isFinite(latest) ? latest : (times.at(-1) ?? Date.now());
  return {
    times,
    step: stats.step * 1000,
    domain: [end - span, end],
    ...(live ? fromSamples(samples, stats.step) : fromPoints(points)),
  };
}
