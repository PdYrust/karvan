import { describe, expect, it } from 'vitest';
import type { Sample, ServerStats } from '../../lib/api/analytics';
import { liveWindow, mergeLive, serverView } from './server-view';

const t0 = Date.UTC(2026, 9, 17, 12, 0, 0);

function sample(seq: number, cores = [0.2, 0.4]): Sample {
  return {
    seq,
    at: new Date(t0 + seq * 5000).toISOString(),
    cpu: 0.3,
    cores,
    load: [1, 0.5, 0.25],
    mem: { total: 100, used: 40, cache: 30, available: 60, swapTotal: 10, swapUsed: 2 },
    netRx: 10,
    netTx: 5,
    ifaces: [],
    diskRead: 1,
    diskWrite: 2,
    tcp: 7,
    tcpTimeWait: 3,
    procs: 100,
    procsRunning: 2,
    uptime: 1000,
  };
}

function stats(samples: Sample[], extra: Partial<ServerStats> = {}): ServerStats {
  const now = samples.at(-1) ?? null;
  return {
    available: now !== null,
    range: 'live',
    step: 5,
    seq: now?.seq ?? 0,
    facts: {
      hostname: 'h',
      os: '',
      kernel: '',
      arch: 'amd64',
      cpuModel: '',
      cores: 2,
      bootedAt: '',
      karvanRss: 0,
      startedAt: '',
    },
    disks: [],
    now,
    samples,
    ...extra,
  };
}

describe('mergeLive', () => {
  it('appends only samples it does not have yet', () => {
    const prev = stats([sample(1), sample(2)]);
    const merged = mergeLive(prev, stats([sample(2), sample(3)]));
    expect(merged.samples?.map((s) => s.seq)).toEqual([1, 2, 3]);
    expect(merged.seq).toBe(3);
  });

  it('starts over when Karvan restarted and its seq went back', () => {
    const prev = stats([sample(40), sample(41)]);
    expect(mergeLive(prev, stats([sample(1)])).samples?.map((s) => s.seq)).toEqual([1]);
  });

  it('keeps 15 minutes', () => {
    const many = Array.from({ length: 200 }, (_, i) => sample(i + 1));
    const merged = mergeLive(undefined, stats(many));
    const kept = merged.samples ?? [];
    expect(kept.length).toBe(liveWindow / 5000 + 1);
    expect(kept.at(-1)?.seq).toBe(200);
  });
});

describe('serverView', () => {
  it('draws live samples up to the latest one, with each core', () => {
    const v = serverView(stats([sample(1), sample(2)]), liveWindow);
    expect(v.live).toBe(true);
    expect(v.times).toEqual([t0 + 5000, t0 + 10_000]);
    expect(v.domain).toEqual([t0 + 10_000 - liveWindow, t0 + 10_000]);
    expect(v.step).toBe(5000);
    expect(v.cores).toEqual([
      [0.2, 0.2],
      [0.4, 0.4],
    ]);
    expect(v.busiest).toEqual([]);
    expect(v.load15).toEqual([0.25, 0.25]);
  });

  it('keeps only the busiest core of a large server', () => {
    const v = serverView(stats([sample(1, [0.1, 0.9, 0.2, 0.3, 0.4])]), liveWindow);
    expect(v.cores).toEqual([]);
    expect(v.busiest).toEqual([0.9]);
  });

  it('draws minutes with their peaks over a range', () => {
    const minute = {
      t: t0 / 1000,
      cpu: 0.2,
      cpuMax: 0.7,
      load1: 1,
      memUsed: 1,
      memCache: 2,
      memTotal: 4,
      swapUsed: 0,
      netRx: 3,
      netTx: 4,
      netRxMax: 9,
      netTxMax: 9,
      diskRead: 0,
      diskWrite: 0,
      tcp: 5,
      procs: 9,
      minutes: 3,
    };
    const v = serverView(
      stats([sample(3)], { range: '24h', step: 300, samples: undefined, points: [minute] }),
      24 * 3_600_000,
    );
    expect(v.live).toBe(false);
    expect(v.times).toEqual([t0]);
    expect(v.cpuPeak).toEqual([0.7]);
    expect(v.step).toBe(300_000);
    expect(v.domain[1]).toBe(t0 + 15_000);
    // A 5-minute bucket the server saw for 3 minutes stands for 180 seconds.
    expect(v.covered).toEqual([180]);
  });

  it('counts each live sample as its 5 seconds', () => {
    expect(serverView(stats([sample(1), sample(2)]), liveWindow).covered).toEqual([5, 5]);
  });
});
