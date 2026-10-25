import { afterEach, describe, expect, it, vi } from 'vitest';
import { curvesFor, maxDots, travelMs } from './dots';
import { createDots } from './use-dots';

const lanes = { forward: 'a', back: 'b', sink: 'c' } as const;

function frames() {
  const queue: FrameRequestCallback[] = [];
  vi.stubGlobal(
    'requestAnimationFrame',
    vi.fn((cb: FrameRequestCallback) => queue.push(cb)),
  );
  vi.stubGlobal('cancelAnimationFrame', vi.fn());
  /** Runs the waiting frame at time t; false when none waits. */
  return (t: number) => {
    const cb = queue.shift();
    cb?.(t);
    return cb !== undefined;
  };
}

function hidden(value: boolean) {
  Object.defineProperty(document, 'hidden', { value, configurable: true });
}

afterEach(() => {
  vi.unstubAllGlobals();
  hidden(false);
});

describe('createDots', () => {
  it('runs a frame only once dots are sent, and stops when they have landed', () => {
    const run = frames();
    const dots = createDots();
    dots.configure(curvesFor(300, 144, false), lanes);
    expect(run(0)).toBe(false);
    vi.spyOn(performance, 'now').mockReturnValue(0);
    dots.send(3, 'forward');
    expect(dots.inFlight()).toBe(3);
    let t = 0;
    while (run(t)) t += 40;
    expect(t).toBeGreaterThan(travelMs);
    expect(dots.inFlight()).toBe(0);
  });

  it('draws nothing while the tab is hidden', () => {
    const run = frames();
    const dots = createDots();
    dots.configure(curvesFor(300, 144, false), lanes);
    hidden(true);
    dots.send(2, 'forward');
    expect(run(0)).toBe(false);
  });

  it('never has more than the cap in flight', () => {
    frames();
    const dots = createDots();
    dots.send(maxDots + 10, 'forward');
    expect(dots.inFlight()).toBe(maxDots);
  });

  it('keeps a steady lane busy until it is turned off', () => {
    const run = frames();
    const dots = createDots();
    dots.configure(curvesFor(300, 144, false), lanes, 'forward');
    for (let t = 0; t < 2000; t += 40) expect(run(t)).toBe(true);
    expect(dots.inFlight()).toBeGreaterThan(1);
    dots.configure(curvesFor(300, 144, false), lanes);
    let t = 2000;
    while (run(t)) t += 40;
    expect(dots.inFlight()).toBe(0);
  });
});
