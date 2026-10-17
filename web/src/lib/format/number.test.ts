import { describe, expect, it } from 'vitest';
import { formatDuration } from './duration';
import {
  formatBytes,
  formatCompact,
  formatNumber,
  formatPercent,
  formatRate,
  splitBytes,
} from './number';

describe('numbers', () => {
  it('groups digits in each language', () => {
    expect(formatNumber(12500, 'en')).toBe('12,500');
    expect(formatNumber(12500, 'fa')).toBe('۱۲٬۵۰۰');
    expect(formatNumber(2.345, 'en', 1)).toBe('2.3');
  });

  it('shortens large counts', () => {
    expect(formatCompact(5830, 'en')).toBe('5.8K');
    expect(formatCompact(42, 'en')).toBe('42');
  });

  it('formats ratios as percent', () => {
    expect(formatPercent(0.565, 'en')).toBe('56.5%');
    expect(formatPercent(1, 'fa')).toBe('۱۰۰٪');
  });
});

describe('bytes', () => {
  it('steps by 1024 like the panels', () => {
    expect(splitBytes(1024 ** 3 * 1.5)).toEqual({ value: 1.5, unit: 'GB' });
    expect(splitBytes(512)).toEqual({ value: 512, unit: 'B' });
    expect(splitBytes(-2048)).toEqual({ value: -2, unit: 'KB' });
  });

  it('keeps one decimal only under ten', () => {
    expect(formatBytes(1024 ** 3 * 1.54, 'en')).toBe('1.5 GB');
    expect(formatBytes(1024 ** 3 * 12.6, 'en')).toBe('13 GB');
    expect(formatBytes(0, 'en')).toBe('0 B');
    expect(formatBytes(1024 ** 2 * 3.5, 'fa')).toBe('۳٫۵ MB');
    expect(formatRate(1024 * 200, 'en')).toBe('200 KB/s');
  });

  it('stops at the largest unit', () => {
    expect(formatBytes(1024 ** 6 * 3, 'en')).toBe('3,072 PB');
  });
});

describe('durations', () => {
  it('names the two largest units', () => {
    expect(formatDuration(3 * 86400 + 4 * 3600 + 5, 'en')).toBe('3 days 4 hours');
    expect(formatDuration(3661, 'en')).toBe('1 hour 1 minute');
    expect(formatDuration(3 * 86400 + 4 * 3600, 'fa')).toBe('۳ روز و ۴ ساعت');
  });

  it('drops a zero in the middle', () => {
    expect(formatDuration(3 * 86400 + 300, 'en')).toBe('3 days');
    expect(formatDuration(60, 'en')).toBe('1 minute');
  });

  it('counts seconds when that is all there is', () => {
    expect(formatDuration(0, 'en')).toBe('0 seconds');
    expect(formatDuration(45.9, 'en')).toBe('45 seconds');
    expect(formatDuration(-5, 'en')).toBe('0 seconds');
    expect(formatDuration(90061, 'en', 3)).toBe('1 day 1 hour 1 minute');
  });
});
