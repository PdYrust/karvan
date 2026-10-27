import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useLingering } from './use-lingering';
// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement;
let root: Root;
function Probe({ value }: { value: string | null }) {
  return useLingering(value, 200) ?? 'none';
}
const show = (value: string | null) => act(() => root.render(<Probe value={value} />));
const wait = (ms: number) => act(() => vi.advanceTimersByTime(ms));
beforeEach(() => {
  vi.useFakeTimers();
  host = document.createElement('div');
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  vi.useRealTimers();
});
describe('useLingering', () => {
  it('keeps the last value while closing, then lets it go', () => {
    show('a');
    show(null);
    expect(host.textContent).toBe('a');
    wait(199);
    expect(host.textContent).toBe('a');
    wait(1);
    expect(host.textContent).toBe('none');
  });
  it('shows a new value at once, even while closing', () => {
    show('a');
    show(null);
    show('b');
    expect(host.textContent).toBe('b');
    wait(500);
    expect(host.textContent).toBe('b');
  });
  it('starts empty', () => {
    show(null);
    expect(host.textContent).toBe('none');
  });
});
