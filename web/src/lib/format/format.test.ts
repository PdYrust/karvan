import { describe, expect, it } from 'vitest';
import { formatShortDate, zoneLabel } from './date';
import { makeFormat } from './format';

// 2026-10-15 15:10 UTC is 18:40 in Tehran.
const moment = new Date(Date.UTC(2026, 9, 15, 15, 10));

describe('makeFormat', () => {
  it('binds language, zone and calendar', () => {
    const fa = makeFormat('fa', { timeZone: 'Asia/Tehran', calendar: 'auto' });
    expect(fa.time(moment)).toBe('۱۸:۴۰');
    expect(fa.longDate(moment)).toBe('پنجشنبه ۲۳ مهر ۱۴۰۵');
    expect(fa.zone).toBe('تهران');
    const en = makeFormat('en', { timeZone: 'UTC', calendar: 'persian' });
    expect(en.time(moment)).toBe('15:10');
    expect(en.longDate(moment)).toContain('1405');
    expect(en.bytes(1536)).toBe('1.5 KB');
    expect(en.duration(90)).toBe('1 minute 30 seconds');
    expect(en.number(1234)).toBe('1,234');
    expect(en.compact(1500)).toBe('1.5K');
    expect(en.percent(0.25)).toBe('25%');
    expect(en.rate(2048)).toBe('2 KB/s');
    expect(en.dateTime(moment)).toContain('15:10');
    expect(en.relative(moment, new Date(moment.getTime() + 60_000))).toBe('1 minute ago');
  });

  it('writes short dates for axes', () => {
    expect(formatShortDate(moment, 'en')).toBe('Oct 15');
    expect(formatShortDate(moment, 'fa')).toBe('۲۳ مهر');
  });

  it('names zones', () => {
    expect(zoneLabel('Asia/Tehran', 'en')).toBe('Tehran');
    expect(zoneLabel('Europe/Berlin', 'en')).toMatch(/Europe|Germany/);
  });
});
