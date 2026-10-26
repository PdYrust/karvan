import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { MenuIcon } from './MenuIcon';
describe('MenuIcon', () => {
  it('draws the short line from the start edge and mirrors in RTL', () => {
    const host = document.createElement('div');
    host.innerHTML = renderToStaticMarkup(<MenuIcon />);
    const svg = host.querySelector('svg');
    expect(svg?.getAttribute('class')).toContain('rtl:-scale-x-100');
    expect(svg?.getAttribute('aria-hidden')).toBe('true');
    // Two full lines, then a short one that starts where they start (the left, which is the start in LTR).
    expect(host.querySelector('path')?.getAttribute('d')).toBe('M4 6h16M4 12h16M4 18h9');
  });
});
