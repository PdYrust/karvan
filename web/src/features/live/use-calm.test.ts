import { afterEach, describe, expect, it, vi } from 'vitest';
import { calmNow } from './use-calm';

const media = (reduce: boolean) => (q: string) =>
  ({ matches: reduce && q.includes('reduce'), media: q }) as MediaQueryList;

afterEach(() => {
  vi.unstubAllGlobals();
  Reflect.deleteProperty(navigator, 'connection');
});

describe('calmNow', () => {
  it('moves the dots when nothing asks for calm', () => {
    vi.stubGlobal('matchMedia', media(false));
    expect(calmNow()).toBe(false);
  });

  it('stays still when the operator asks for less motion', () => {
    vi.stubGlobal('matchMedia', media(true));
    expect(calmNow()).toBe(true);
  });

  it('stays still when the browser saves data', () => {
    vi.stubGlobal('matchMedia', media(false));
    Object.defineProperty(navigator, 'connection', {
      value: { saveData: true },
      configurable: true,
    });
    expect(calmNow()).toBe(true);
  });
});
