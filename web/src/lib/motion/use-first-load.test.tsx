import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useFirstLoad, type FirstLoad } from './use-first-load';

// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement;
let root: Root;
let reduce = false;

function Probe({ value }: { value: string | undefined }) {
  return JSON.stringify(useFirstLoad(value));
}

/** What the hook returned on the last render; JSON drops an undefined `data`. */
const seen = (): FirstLoad<string> => ({ data: undefined, ...JSON.parse(host.textContent) });

const show = (value: string | undefined) => act(() => root.render(<Probe value={value} />));
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

describe('useFirstLoad', () => {
  it('keeps the skeleton for the minimum time once shown, then reveals the data', () => {
    show(undefined);
    expect(seen()).toEqual({ data: undefined, holding: false, reveal: false });
    wait(50);
    show('rows');
    expect(seen()).toEqual({ data: undefined, holding: true, reveal: false });
    wait(300);
    expect(seen().data).toBeUndefined();
    wait(60);
    expect(seen()).toEqual({ data: 'rows', holding: false, reveal: true });
  });

  it('does not hold data that arrives after the minimum', () => {
    show(undefined);
    wait(600);
    show('rows');
    wait(0);
    expect(seen()).toEqual({ data: 'rows', holding: false, reveal: true });
  });

  it('passes cached data, refetches and polls straight through', () => {
    show('cached');
    expect(seen()).toEqual({ data: 'cached', holding: false, reveal: false });
    show('polled');
    expect(seen()).toEqual({ data: 'polled', holding: false, reveal: false });
  });

  it('holds the skeleton again when the data goes away', () => {
    show('first');
    show(undefined);
    expect(seen().data).toBeUndefined();
    show('second');
    expect(seen().data).toBeUndefined();
    wait(400);
    expect(seen()).toEqual({ data: 'second', holding: false, reveal: true });
  });

  it('holds nothing and fades nothing with reduced motion', () => {
    reduce = true;
    show(undefined);
    show('rows');
    expect(seen()).toEqual({ data: 'rows', holding: false, reveal: false });
    wait(400);
    expect(seen()).toEqual({ data: 'rows', holding: false, reveal: false });
  });
});
