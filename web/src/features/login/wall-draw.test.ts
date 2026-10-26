import { describe, expect, it } from 'vitest';
import { depthTone, tone, type Palette } from './wall-draw';

describe('depthTone', () => {
  it('keeps a near face as it is', () => {
    expect(depthTone(0, 0)).toBe(0);
    expect(depthTone(0.6, 0)).toBeCloseTo(0.6);
  });

  it('sinks far faces into the shade with less contrast between them', () => {
    const near = depthTone(0.7, 0) - depthTone(0, 0);
    const far = depthTone(0.7, 1) - depthTone(0, 1);
    expect(depthTone(0, 1)).toBeGreaterThan(depthTone(0, 0));
    expect(far).toBeGreaterThan(0);
    expect(far).toBeLessThan(near / 2);
  });
});

describe('tone', () => {
  const p: Palette = { light: 'L', toShade: ['C', 'S1', 'S2'], toLight: ['C', 'L1', 'L2'] };

  it('mixes toward the shade above zero and toward the light below it', () => {
    expect(tone(p, 0)).toBe('C');
    expect(tone(p, 0.5)).toBe('S1');
    expect(tone(p, -1)).toBe('L2');
  });

  it('stops at the ends of the ramps', () => {
    expect(tone(p, 3)).toBe('S2');
    expect(tone(p, -3)).toBe('L2');
  });
});
