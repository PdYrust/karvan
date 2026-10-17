import { describe, expect, it } from 'vitest';
import { createTranslate } from '../../../i18n/core';
import type { Sample } from '../../../lib/api/analytics';
import { makeFormat } from '../../../lib/format/format';
import { analyticsMessages } from '../messages';
import type { ServerView } from '../server-view';
import { mean, serverCharts, total } from './charts';

const t = createTranslate(analyticsMessages.en, 'en');
const fmt = makeFormat('en', { timeZone: 'UTC', calendar: 'auto' });

const now: Sample = {
  seq: 9,
  at: '2026-10-17T12:00:00Z',
  cpu: 0.42,
  cores: [0.3, 0.54],
  load: [1.5, 1.25, 1],
  mem: { total: 8, used: 4, cache: 2, available: 4, swapTotal: 0, swapUsed: 0 },
  netRx: 2048,
  netTx: 1024,
  ifaces: [],
  diskRead: 0,
  diskWrite: 512,
  tcp: 30,
  tcpTimeWait: 4,
  procs: 200,
  procsRunning: 3,
  uptime: 60,
};

function view(over: Partial<ServerView>): ServerView {
  return {
    times: [0, 1],
    step: 300_000,
    domain: [0, 1],
    live: false,
    cpu: [0.2, 0.4],
    cpuPeak: [0.5, 0.9],
    cores: [],
    busiest: [],
    load1: [1, 2],
    load5: [],
    load15: [],
    memUsed: [1, 3],
    memCache: [1, 1],
    swap: [0, 0],
    netRx: [1024, 1024],
    netTx: [0, 512],
    diskRead: [0, 0],
    diskWrite: [0, 0],
    tcp: [10, 20],
    closing: [],
    covered: [300, 60],
    ...over,
  };
}

const legend = (v: ServerView, key: string) =>
  serverCharts(v, now, t, fmt)
    .find((c) => c.key === key)
    ?.series.map((s) => `${s.name} ${s.value ?? '-'}`);

describe('mean and total', () => {
  it('skips missing values', () => {
    expect(mean([1, null, 3])).toBe(2);
    expect(mean([null, undefined])).toBeUndefined();
  });

  it('weighs each rate by the seconds it stands for', () => {
    expect(total([10, null, 2], [60, 60, 30])).toBe(660);
  });
});

describe('serverCharts', () => {
  it('shows the latest sample when live, with each core', () => {
    const live = view({ live: true, cores: [[0.3], [0.54]], load5: [1], load15: [1] });
    expect(legend(live, 'cpu')).toEqual(['All cores 42%', 'Core 1 30%', 'Core 2 54%']);
    expect(legend(live, 'network')).toEqual(['In 2 KB/s', 'Out 1 KB/s']);
    expect(legend(live, 'tcp')).toEqual(['Open 30', 'Closing 4']);
  });

  it('shows the average and the peak over a range, and what the network moved', () => {
    const range = view({});
    expect(legend(range, 'cpu')).toEqual(['Average 30%', 'Peak 90%']);
    // 1 KB/s for 300 s and for 60 s.
    expect(legend(range, 'network')).toEqual(['In 360 KB', 'Out 30 KB']);
    expect(legend(range, 'tcp')).toEqual(['Open 15']);
  });

  it('marks the busiest core of a large server as a second reading', () => {
    const big = view({ live: true, busiest: [0.9] });
    const cpu = serverCharts(big, now, t, fmt).find((c) => c.key === 'cpu');
    expect(cpu?.series.at(-1)).toMatchObject({ name: 'Busiest core', dashed: true });
  });
});
