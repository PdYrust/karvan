import { describe, expect, it } from 'vitest';
import { nav } from './nav';
import { builtPages, gotoTargets } from './routes-built';

describe('built routes', () => {
  it('gives every sidebar item a route and its own g letter', () => {
    const items = nav.flatMap((g) => g.items);
    expect(builtPages()).toEqual(items);
    expect(new Set(items.map((i) => i.key)).size).toBe(items.length);
    expect(items.every((i) => i.to.startsWith('/'))).toBe(true);
  });
  it('opens each page the spec names with g then its letter', () => {
    const targets = gotoTargets();
    expect({
      o: targets.KeyO,
      a: targets.KeyA,
      m: targets.KeyM,
      r: targets.KeyR,
      l: targets.KeyL,
      p: targets.KeyP,
      s: targets.KeyS,
    }).toEqual({
      o: '/',
      a: '/analytics',
      m: '/migrations',
      r: '/reports',
      l: '/links',
      p: '/panels',
      s: '/settings',
    });
  });
});
