import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Switch } from './Switch';
// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
});
describe('Switch', () => {
  it('mirrors in right to left text so on puts the thumb on the left', () => {
    host.dir = 'rtl';
    act(() => root.render(<Switch aria-label="x" checked onCheckedChange={vi.fn()} />));
    const track = host.querySelector('[role="switch"]');
    expect(track?.getAttribute('aria-checked')).toBe('true');
    // Kumo slides the thumb with physical left offsets; the rtl variant flips
    // the track so the thumb travels toward the inline end.
    expect(track?.className.split(' ')).toContain('rtl:-scale-x-100');
    expect(track?.firstElementChild?.className).toContain('left-4.5');
  });
});
