import { Toasty } from '@cloudflare/kumo/components/toast';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { overviewKey } from '../../features/overview/use-overview';
import { sessionKey } from '../../features/session/use-session';
import { LocaleProvider } from '../../i18n/locale';
import { RegionProvider } from '../../lib/preferences/region';
import { ThemeProvider } from '../../lib/preferences/theme';
import { AccountMenu } from './AccountMenu';
import { GitHubLink } from './GitHubLink';

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

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
  document.documentElement.lang = '';
});

async function settle() {
  await act(async () => {
    await new Promise((r) => setTimeout(r, 50));
  });
}

function renderHeader(overview: { version: string; latestVersion?: string }) {
  const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } });
  client.setQueryData(sessionKey, {
    authenticated: true,
    operator: { username: 'admin', totpEnabled: false },
    csrfToken: 'tok',
  });
  client.setQueryData(overviewKey, {
    ...overview,
    safeMode: true,
    security: { totpEnabled: false, previousSignIn: null, tls: 'on' },
    migrations: { jobs: {}, moved: 0, skipped: 0, lastEnded: null },
    activity: [],
  });
  act(() =>
    root.render(
      <ThemeProvider>
        <LocaleProvider>
          <RegionProvider>
            <QueryClientProvider client={client}>
              <Toasty>
                <MemoryRouter>
                  <GitHubLink />
                  <AccountMenu onShortcuts={() => undefined} />
                </MemoryRouter>
              </Toasty>
            </QueryClientProvider>
          </RegionProvider>
        </LocaleProvider>
      </ThemeProvider>,
    ),
  );
}

describe('AccountMenu and header links', () => {
  it('shows the Telegram and GitHub links in the header and the current version card under the username', async () => {
    renderHeader({ version: '0.5.15-beta' });
    const tg = host.querySelector<HTMLAnchorElement>('a[href="https://t.me/PdYrust"]');
    const gh = host.querySelector<HTMLAnchorElement>('a[href="https://github.com/PdYrust/karvan"]');
    expect(tg?.getAttribute('aria-label')).toBe('Karvan on Telegram');
    expect(gh?.getAttribute('aria-label')).toBe('Karvan on GitHub');

    const trigger = host.querySelector<HTMLButtonElement>('button[aria-label="Account"]');
    expect(trigger?.querySelector('.bg-kumo-success')).toBeNull();
    act(() => trigger?.click());
    await settle();

    const menu = document.querySelector('[role="menu"]');
    expect(menu?.textContent).toContain('admin');
    expect(menu?.textContent).toContain('0.5.15-beta');
    expect(menu?.textContent).toContain('Up to date');
  });

  it('adds a green dot on the profile button and shows the new release under the current version in the card', async () => {
    renderHeader({ version: '0.5.15-beta', latestVersion: '0.5.16-beta' });
    const trigger = host.querySelector<HTMLButtonElement>('button[aria-label="Account"]');
    expect(trigger?.querySelector('.bg-kumo-success')).toBeTruthy();
    act(() => trigger?.click());
    await settle();

    const menu = document.querySelector('[role="menu"]');
    const link = menu?.querySelector<HTMLAnchorElement>('a[href*="releases/tag/v0.5.16-beta"]');
    expect(link).toBeTruthy();
    expect(link?.textContent).toContain('0.5.15-beta');
    expect(link?.textContent).toContain('0.5.16-beta');
    expect(link?.textContent).toContain('available');
  });
});
