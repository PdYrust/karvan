import { act, useEffect } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { MemoryRouter, useLocation } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { TabSwitcher } from './TabSwitcher';

// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

const tabs = [
  { to: '/analytics/server?range=7d', label: 'Server' },
  { to: '/analytics/panels/2?range=7d', label: 'pasarguard-b' },
];
let host: HTMLDivElement;
let root: Root;
let at = '';
function Where() {
  const { pathname, search } = useLocation();
  useEffect(() => {
    at = pathname + search;
  });
  return null;
}
function phone(matches: boolean) {
  vi.stubGlobal('matchMedia', () => ({
    matches,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  }));
}
function render() {
  act(() =>
    root.render(
      <MemoryRouter initialEntries={['/analytics/server?range=7d']}>
        <TabSwitcher label="Sections" tabs={tabs} icon={null} />
        <Where />
      </MemoryRouter>,
    ),
  );
}
async function settle() {
  await act(async () => {
    await new Promise((r) => setTimeout(r, 50));
  });
}
beforeEach(() => {
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
  vi.unstubAllGlobals();
});

describe('TabSwitcher', () => {
  it('on a phone shows one button that opens the tab list and keeps the range', async () => {
    phone(true);
    render();
    const trigger = host.querySelector<HTMLButtonElement>('button[aria-haspopup]');
    expect(trigger?.getAttribute('aria-label')).toBe('Sections: Server');
    act(() => trigger?.click());
    await settle();
    const items = [...document.querySelectorAll<HTMLElement>('[role="menuitemradio"]')];
    expect(items.map((el) => el.textContent)).toEqual(['Server', 'pasarguard-b']);
    expect(items[0]?.getAttribute('aria-checked')).toBe('true');
    act(() => items[1]?.click());
    await settle();
    expect(at).toBe('/analytics/panels/2?range=7d');
    expect(trigger?.getAttribute('aria-label')).toBe('Sections: pasarguard-b');
  });
  it('shows no visible segments on a phone', () => {
    phone(true);
    render();
    // Only the measuring copy, which is hidden from people and assistive tech.
    const navs = [...host.querySelectorAll('nav')];
    expect(navs).toHaveLength(1);
    expect(navs[0]?.closest('[aria-hidden="true"]')).toBeTruthy();
  });
});
