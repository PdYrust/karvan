import { afterEach, describe, expect, it, vi } from 'vitest';
import { withThemeTransition } from './theme-transition';

const media = (reduce: boolean) =>
  vi.stubGlobal('matchMedia', (q: string) => ({ matches: reduce && q.includes('reduce') }));

afterEach(() => vi.unstubAllGlobals());

describe('withThemeTransition', () => {
  it('crossfades where the browser has View Transitions', () => {
    media(false);
    const apply = vi.fn();
    const start = vi.fn((update: () => void) => update());
    const doc = { startViewTransition: start } as unknown as Document;
    expect(withThemeTransition(apply, doc)).toBe('faded');
    expect(start).toHaveBeenCalledOnce();
    expect(apply).toHaveBeenCalledOnce();
  });

  it('changes at once where the browser has none', () => {
    media(false);
    const apply = vi.fn();
    expect(withThemeTransition(apply, {} as Document)).toBe('instant');
    expect(apply).toHaveBeenCalledOnce();
  });

  it('changes at once with reduced motion', () => {
    media(true);
    const apply = vi.fn();
    const start = vi.fn();
    const doc = { startViewTransition: start } as unknown as Document;
    expect(withThemeTransition(apply, doc)).toBe('instant');
    expect(start).not.toHaveBeenCalled();
    expect(apply).toHaveBeenCalledOnce();
  });
});
