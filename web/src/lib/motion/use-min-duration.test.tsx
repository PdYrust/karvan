import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { motionLeft, useMinDuration } from './use-min-duration';

// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement;
let root: Root;
let reduce = false;

function Probe({ active, step }: { active: boolean; step?: number }) {
  return String(useMinDuration(active, 500, step));
}

const seen = () => host.textContent === 'true';
const show = (active: boolean, step?: number) =>
  act(() => root.render(<Probe active={active} step={step} />));
const wait = (ms: number) => act(() => vi.advanceTimersByTime(ms));

beforeEach(() => {
  vi.useFakeTimers();
  reduce = false;
  vi.stubGlobal('matchMedia', (q: string) => ({
    matches: reduce && q.includes('reduce'),
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  }));
  host = document.createElement('div');
  root = createRoot(host);
});

afterEach(() => {
  act(() => root.unmount());
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe('motionLeft', () => {
  it('runs at least the minimum', () => {
    expect(motionLeft(0, 150)).toBe(150);
    expect(motionLeft(100, 150)).toBe(50);
    expect(motionLeft(400, 150)).toBe(0);
  });

  it('ends on a whole step, never before the first', () => {
    expect(motionLeft(0, 500, 500)).toBe(500);
    expect(motionLeft(20, 500, 500)).toBe(480);
    expect(motionLeft(500, 500, 500)).toBe(0);
    expect(motionLeft(620, 500, 500)).toBe(380);
  });
});

describe('useMinDuration', () => {
  it('holds an instant answer until the minimum has run', () => {
    show(false);
    expect(seen()).toBe(false);
    show(true);
    show(false);
    expect(seen()).toBe(true);
    wait(490);
    expect(seen()).toBe(true);
    wait(20);
    expect(seen()).toBe(false);
  });

  it('lets a long answer end on the next whole turn', () => {
    show(true, 500);
    wait(700);
    show(false, 500);
    expect(seen()).toBe(true);
    wait(290);
    expect(seen()).toBe(true);
    wait(20);
    expect(seen()).toBe(false);
  });

  it('holds nothing with reduced motion', () => {
    reduce = true;
    show(true);
    show(false);
    expect(seen()).toBe(false);
  });
});
