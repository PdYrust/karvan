import { describe, expect, it } from 'vitest';
import { zoneItems } from './zones';
describe('zoneItems', () => {
  it('names each zone with its offset, Tehran first', () => {
    const items = zoneItems('Asia/Tehran', 'en');
    expect(items[0]).toEqual({ id: 'Asia/Tehran', label: 'Tehran (GMT+3:30)' });
    expect(items.map((z) => z.id)).toContain('UTC');
    expect(zoneItems('Asia/Tehran', 'fa')[0]?.label).toMatch(/^تهران /);
  });
  it('offers a zone chosen elsewhere at the top', () => {
    const items = zoneItems('Asia/Kabul', 'en');
    expect(items[0]?.id).toBe('Asia/Kabul');
    expect(items.filter((z) => z.id === 'Asia/Kabul')).toHaveLength(1);
  });
});
