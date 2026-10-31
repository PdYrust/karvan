import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { CurrentPanelProvider } from '../../features/panels/current-panel';
import { panelsKey } from '../../features/panels/use-panels';
import { LocaleProvider } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { PanelSwitcher } from './PanelSwitcher';
// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
const panels = [{ id: 2, name: 'pasarguard-b', kind: 'pasarguard', ready: true }] as Panel[];
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  document.documentElement.lang = 'en';
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
  localStorage.clear();
});
async function settle() {
  await act(async () => {
    await new Promise((r) => setTimeout(r, 50));
  });
}
async function openItems() {
  const client = new QueryClient();
  client.setQueryData(panelsKey, panels);
  act(() =>
    root.render(
      <QueryClientProvider client={client}>
        <LocaleProvider>
          <CurrentPanelProvider>
            <PanelSwitcher />
          </CurrentPanelProvider>
        </LocaleProvider>
      </QueryClientProvider>,
    ),
  );
  act(() => host.querySelector<HTMLButtonElement>('button[aria-haspopup]')?.click());
  await settle();
  return [...document.querySelectorAll<HTMLElement>('[role="menuitemradio"]')].map((el) => ({
    text: el.textContent ?? '',
    checked: el.getAttribute('aria-checked') === 'true',
  }));
}
describe('PanelSwitcher', () => {
  it('lists all panels and every panel when all panels is selected', async () => {
    const items = await openItems();
    expect(items.map((i) => i.text)).toEqual(['All panels', 'pasarguard-b']);
    expect(items.map((i) => i.checked)).toEqual([true, false]);
  });
  it('lists all panels and every panel when a panel is selected', async () => {
    localStorage.setItem('karvan.panel', '2');
    const items = await openItems();
    expect(items.map((i) => i.text)).toEqual(['All panels', 'pasarguard-b']);
    expect(items.map((i) => i.checked)).toEqual([false, true]);
  });
});
