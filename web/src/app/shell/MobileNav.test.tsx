import { Toasty } from '@cloudflare/kumo/components/toast';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { CurrentPanelProvider } from '../../features/panels/current-panel';
import { panelsKey } from '../../features/panels/use-panels';
import { LocaleProvider } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { MobileNav } from './MobileNav';
// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
const panels = [
  { id: 1, name: 'marzban-a', kind: 'marzban', ready: true },
  { id: 2, name: 'pasarguard-b', kind: 'pasarguard', ready: true },
] as Panel[];
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  localStorage.setItem('karvan.lang', 'en');
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
describe('MobileNav', () => {
  it('opens the panel switcher inside the sheet and switches the panel', async () => {
    const client = new QueryClient();
    client.setQueryData(panelsKey, panels);
    act(() =>
      root.render(
        <QueryClientProvider client={client}>
          <LocaleProvider>
            <Toasty>
              <MemoryRouter>
                <CurrentPanelProvider>
                  <MobileNav open onClose={() => undefined}>
                    <p>nav</p>
                  </MobileNav>
                </CurrentPanelProvider>
              </MemoryRouter>
            </Toasty>
          </LocaleProvider>
        </QueryClientProvider>,
      ),
    );
    const dialog = document.querySelector('dialog');
    const trigger = dialog?.querySelector<HTMLButtonElement>('button[aria-haspopup]');
    expect(trigger).toBeTruthy();
    act(() => trigger?.click());
    await settle();
    // A modal sheet is in the top layer with the page behind it inert: the menu must live inside it.
    const menu = document.querySelector('[role="menu"]');
    expect(menu).toBeTruthy();
    expect(dialog?.contains(menu ?? null)).toBe(true);
    const item = [...(menu?.querySelectorAll<HTMLElement>('[role="menuitemradio"]') ?? [])].find(
      (el) => el.textContent?.includes('pasarguard-b'),
    );
    expect(item).toBeTruthy();
    act(() => item?.click());
    await settle();
    expect(trigger?.getAttribute('aria-label')).toContain('pasarguard-b');
  });
});
