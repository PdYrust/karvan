import { describe, expect, it } from 'vitest';
import { isDateStep, timeTicks, zoneOffset } from './time';

const minute = 60_000;
const hour = 60 * minute;

describe('zoneOffset', () => {
  it('knows Tehran is three and a half hours ahead', () => {
    expect(zoneOffset('Asia/Tehran', Date.UTC(2026, 9, 17, 12, 0, 30, 250))).toBe(3.5 * hour);
  });

  it('follows summer time where there is one', () => {
    expect(zoneOffset('Europe/Berlin', Date.UTC(2026, 6, 1))).toBe(2 * hour);
    expect(zoneOffset('Europe/Berlin', Date.UTC(2026, 0, 1))).toBe(hour);
    expect(zoneOffset('UTC', Date.UTC(2026, 0, 1))).toBe(0);
  });
});

describe('timeTicks', () => {
  it('lands on round minutes for the live view', () => {
    const to = Date.UTC(2026, 9, 17, 12, 7, 40);
    const { ticks, step } = timeTicks(to - 15 * minute, to, 'UTC');
    expect(step).toBe(5 * minute);
    expect(ticks.map((t) => new Date(t).toISOString().slice(11, 16))).toEqual([
      '11:55',
      '12:00',
      '12:05',
    ]);
  });

  it("lands on round hours of the zone's own clock", () => {
    const to = Date.UTC(2026, 9, 17, 12, 0);
    const { ticks, step } = timeTicks(to - 24 * hour, to, 'Asia/Tehran');
    expect(step).toBe(6 * hour);
    // 00:00 in Tehran is 20:30 UTC the day before.
    expect(ticks.map((t) => new Date(t + 3.5 * hour).toISOString().slice(11, 16))).toEqual([
      '18:00',
      '00:00',
      '06:00',
      '12:00',
    ]);
  });

  it('marks midnights over a week, labelled as dates', () => {
    const to = Date.UTC(2026, 9, 17, 9, 0);
    const { ticks, step } = timeTicks(to - 7 * 24 * hour, to, 'UTC');
    expect(isDateStep(step)).toBe(true);
    expect(ticks.length).toBeLessThanOrEqual(5);
    expect(ticks.every((t) => t % (24 * hour) === 0)).toBe(true);
    expect(isDateStep(hour)).toBe(false);
  });
});
