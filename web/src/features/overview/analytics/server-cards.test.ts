import { describe, expect, it } from 'vitest';
import { createTranslate } from '../../../i18n/core';
import type { HostMinute, Sample, ServerStats } from '../../../lib/api/analytics';
import type { Value } from '../../../lib/charts/series';
import { makeFormat } from '../../../lib/format/format';
import { analyticsMessages } from '../../analytics/messages';
import { overviewMessages } from '../messages';
import { serverCards } from './server-cards';

const t = createTranslate(analyticsMessages.en, 'en');
const tOverview = createTranslate(overviewMessages.en, 'en');
const fmt = makeFormat('en', { timeZone: 'UTC', calendar: 'auto' });
const t0 = Date.UTC(2026, 9, 27, 14, 7, 0);

function sample(seq: number, cores: number[]): Sample {
  return {
    seq,
    at: new Date(t0 + seq * 5000).toISOString(),
    cpu: 0.5,
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

/** A minute saved before the restart, starting `ago` minutes before t0. */
function minute(ago: number): HostMinute {
  return {
    t: t0 / 1000 - ago * 60,
    cpu: 0.2,
    cpuMax: 0.4,
    load1: 1,
    memUsed: 30,
    memCache: 20,
    memTotal: 100,
    swapUsed: 1,
    netRx: 3,
    netTx: 4,
    netRxMax: 9,
    netTxMax: 9,
    diskRead: 0,
    diskWrite: 0,
    tcp: 5,
    procs: 9,
    minutes: 1,
  };
}

/** Ten saved minutes (13:57 to 14:06), a missing one inside them, then two samples. */
function stats(cores: number[]): ServerStats {
  const samples = [sample(1, cores), sample(2, cores)];
  return {
    available: true,
    range: 'live',
    step: 5,
    seq: 2,
    facts: {
      hostname: 'h',
      os: '',
      kernel: '',
      arch: 'amd64',
      cpuModel: '',
      cores: cores.length,
      bootedAt: '',
      karvanRss: 0,
      startedAt: '',
    },
    disks: [{ mount: '/', type: 'ext4', total: 10, used: 5, free: 5 }],
    now: samples[1] ?? null,
    samples,
    backfill: [10, 9, 8, 7, 5, 4, 3, 2, 1].map(minute),
  };
}

const has = (v: Value) => v !== null && v !== undefined;
/** The first and last index with a value in any of the lines. */
function span(lines: readonly (readonly Value[])[]): [number, number] {
  const n = Math.max(0, ...lines.map((l) => l.length));
  const at = (i: number) => lines.some((l) => has(l[i]));
  const idx = Array.from({ length: n }, (_, i) => i).filter(at);
  return [idx[0] ?? -1, idx.at(-1) ?? -1];
}

function build(cores: number[]) {
  const s = stats(cores);
  return serverCards(s, s.now as Sample, t, tOverview, fmt);
}

describe('serverCards', () => {
  it.each([
    ['a small server', [0.4, 0.6]],
    ['a large server', [0.1, 0.9, 0.2, 0.3, 0.4]],
  ])('opens a detail over the same times as its card, on %s', (_, cores) => {
    const { cards, view } = build(cores);
    const detailed = cards.filter((c) => c.chart && c.detail);
    expect(detailed.map((c) => c.id)).toEqual(['cpu', 'memory', 'network']);
    for (const c of detailed) {
      const card = span([c.chart?.values ?? []]);
      // Saved minutes first: the card starts at the first one, not at the samples.
      expect(card, c.id).toEqual([0, view.times.length - 1]);
      expect(span(c.detail?.series.map((s) => s.values) ?? []), c.id).toEqual(card);
    }
  });

  it('leads the CPU detail with every core together, the line the card draws', () => {
    const { cards, view } = build([0.4, 0.6]);
    const cpu = cards.find((c) => c.id === 'cpu');
    expect(cpu?.detail?.series.map((s) => s.name)).toEqual(['All cores', 'Core 1', 'Core 2']);
    expect(cpu?.detail?.series[0]?.values).toBe(cpu?.chart?.values);
    expect(cpu?.chart?.values).toEqual([...Array(9).fill(0.2), 0.5, 0.5]);
    // A minute keeps no cores: those lines start with the samples, nothing is made up.
    expect(cpu?.detail?.series[1]?.values).toEqual([...Array(9).fill(null), 0.4, 0.4]);
    // The missing minute stays a hole, and no time is drawn twice at the seam.
    expect(view.spacing?.slice(0, 5)).toEqual([60_000, 60_000, 60_000, 60_000, 60_000]);
    expect((view.times[4] ?? 0) - (view.times[3] ?? 0)).toBe(120_000);
    expect(new Set(view.times).size).toBe(view.times.length);
  });
});
