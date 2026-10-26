import { describe, expect, it } from 'vitest';
import { buildWall, lift, type Column, type Wall } from './wall-geometry';

const frames: [number, number][] = [
  [720, 900],
  [360, 176],
  [1280, 1440],
];

function flatBox(c: Column) {
  const xs = [0, 2, 4, 6, 8, 10].map((i) => c.flat[i] ?? 0);
  const ys = [1, 3, 5, 7, 9, 11].map((i) => c.flat[i] ?? 0);
  return [Math.min(...xs), Math.min(...ys), Math.max(...xs), Math.max(...ys)] as const;
}

const footDistance = (wall: Wall, c: Column) =>
  Math.hypot(c.x - wall.cam.eye[0], c.y - wall.cam.eye[1]);

describe('buildWall', () => {
  it.each(frames)('fills a %i by %i frame to its corners', (w, h) => {
    const wall = buildWall(w, h);
    expect(wall.columns.length).toBeGreaterThan(40);
    for (const [x, y] of [
      [0, 0],
      [w, 0],
      [0, h],
      [w, h],
      [w / 2, h / 2],
    ] as const) {
      const hit = wall.columns.some((c) => {
        const [x0, y0, x1, y1] = flatBox(c);
        return x0 <= x + 1 && x <= x1 + 1 && y0 <= y + 1 && y <= y1 + 1;
      });
      expect(hit).toBe(true);
    }
  });

  it('lists columns from the furthest from the eye to the nearest', () => {
    const wall = buildWall(720, 900);
    const d = wall.columns.map((c) => footDistance(wall, c));
    expect(d.every((v, i) => i === 0 || v <= (d[i - 1] ?? Infinity) + 1e-9)).toBe(true);
  });

  it('shows the sides that face the eye and never the ones that face away', () => {
    const wall = buildWall(720, 900);
    for (const c of wall.columns) {
      const toward = [0, 1, 2, 3, 4, 5].map((k) => {
        const a = ((60 + 60 * k) * Math.PI) / 180;
        return Math.cos(a) * (wall.cam.eye[0] - c.x) + Math.sin(a) * (wall.cam.eye[1] - c.y);
      });
      const most = toward.indexOf(Math.max(...toward));
      const least = toward.indexOf(Math.min(...toward));
      expect(c.sides).toContain(most);
      expect(c.sides).not.toContain(least);
      expect(c.sides.length).toBeLessThanOrEqual(3);
    }
  });

  it('names the neighbour across a side only when it is drawn later', () => {
    const wall = buildWall(720, 900);
    let found = 0;
    wall.columns.forEach((c, n) => {
      c.across.forEach((m) => {
        if (m < 0) return;
        found++;
        const o = wall.columns[m];
        expect(m).toBeGreaterThan(n);
        expect(Math.hypot((o?.x ?? 0) - c.x, (o?.y ?? 0) - c.y)).toBeCloseTo(Math.sqrt(3), 6);
      });
    });
    expect(found).toBeGreaterThan(wall.columns.length);
  });

  it('puts the far corner at the top start', () => {
    const wall = buildWall(720, 900);
    const far = wall.columns.map((c) => c.far);
    expect(Math.min(...far)).toBe(0);
    expect(Math.max(...far)).toBe(1);
    const nearest = (x: number, y: number) =>
      wall.columns.reduce(
        (best, c) => {
          const [x0, y0, x1, y1] = flatBox(c);
          const d = Math.hypot((x0 + x1) / 2 - x, (y0 + y1) / 2 - y);
          return d < best.d ? { d, c } : best;
        },
        { d: Infinity, c: wall.columns[0] },
      ).c;
    expect(nearest(40, 40)?.far ?? 0).toBeGreaterThan(0.8);
    expect(nearest(680, 860)?.far ?? 1).toBeLessThan(0.2);
  });
});

describe('lift', () => {
  const wall = buildWall(720, 900);

  it('stays between settled and fully out, the same for the same moment', () => {
    for (const c of wall.columns.slice(0, 40)) {
      for (let t = 0; t < 60; t += 0.7) {
        const v = lift(c, t);
        expect(v).toBeGreaterThanOrEqual(0);
        expect(v).toBeLessThanOrEqual(1);
        expect(lift(c, t)).toBe(v);
      }
    }
  });

  it('moves slowly, and never all the columns together', () => {
    for (let t = 0; t < 60; t += 1.3) {
      const now = wall.columns.map((c) => lift(c, t));
      const next = wall.columns.map((c) => lift(c, t + 1 / 30));
      const step = Math.max(...now.map((v, i) => Math.abs((next[i] ?? 0) - v)));
      expect(step).toBeLessThan(0.02);
      const mean = now.reduce((a, b) => a + b, 0) / now.length;
      const spread = Math.sqrt(now.reduce((a, b) => a + (b - mean) ** 2, 0) / now.length);
      expect(spread).toBeGreaterThan(0.15);
    }
  });
});
