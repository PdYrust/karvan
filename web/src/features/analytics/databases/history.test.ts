import { describe, expect, it } from 'vitest';
import { bucketSize, day, hour, sumBuckets } from './history';

describe('sumBuckets', () => {
  const from = Date.UTC(2026, 9, 17, 0, 0);
  const to = Date.UTC(2026, 9, 17, 3, 20);

  it('fills quiet hours with zero', () => {
    const b = sumBuckets(
      [
        { at: from + hour, values: [5, 1] },
        { at: from + hour + 60_000, values: [2, 0] },
      ],
      from,
      to,
      hour,
      'UTC',
      2,
    );
    expect(b.times).toEqual([from, from + hour, from + 2 * hour, from + 3 * hour]);
    expect(b.series).toEqual([
      [0, 7, 0, 0],
      [0, 1, 0, 0],
    ]);
    expect(b.step).toBe(hour);
  });

  it("cuts days at the zone's midnight", () => {
    // 21:00 UTC is 00:30 the next day in Tehran.
    const at = Date.UTC(2026, 9, 16, 21, 0);
    const b = sumBuckets([{ at, values: [3] }], at - 2 * day, at, day, 'Asia/Tehran', 1);
    const tehranMidnight = Date.UTC(2026, 9, 16, 20, 30);
    expect(b.times.at(-1)).toBe(tehranMidnight);
    expect(b.series[0]?.at(-1)).toBe(3);
  });

  it('drops values outside the range', () => {
    const b = sumBuckets([{ at: from - day, values: [9] }], from, to, hour, 'UTC', 1);
    expect(b.series[0]?.every((v) => v === 0)).toBe(true);
  });
});

describe('bucketSize', () => {
  it('uses hours up to a day', () => {
    expect(bucketSize(day)).toBe(hour);
    expect(bucketSize(7 * day)).toBe(day);
  });
});
