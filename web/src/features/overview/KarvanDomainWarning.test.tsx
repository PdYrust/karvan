import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { LocaleProvider } from '../../i18n/locale';
import { karvanKeys, type KarvanDomain } from '../../lib/api/karvan';
import { RegionProvider } from '../../lib/preferences/region';
import { KarvanAccess } from '../certificates/KarvanAccess';
import { KarvanDomainWarning } from './KarvanDomainWarning';

// React's act() needs this flag outside a testing library.
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

const base: KarvanDomain = {
  domainId: 3,
  plain: false,
  shared: false,
  only: true,
  access: 'domain',
};

/** The banner's text for the Karvan domain k, and its link. */
function banner(k: KarvanDomain): [string, string | null] {
  const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } });
  client.setQueryData(karvanKeys.all, k);
  act(() =>
    root.render(
      <QueryClientProvider client={client}>
        <LocaleProvider>
          <RegionProvider>
            <MemoryRouter>
              <KarvanDomainWarning />
            </MemoryRouter>
          </RegionProvider>
        </LocaleProvider>
      </QueryClientProvider>,
    ),
  );
  return [host.textContent ?? '', host.querySelector('a')?.getAttribute('href') ?? null];
}

function line(k: KarvanDomain): string {
  act(() =>
    root.render(
      <LocaleProvider>
        <KarvanAccess karvan={k} />
      </LocaleProvider>,
    ),
  );
  return host.textContent ?? '';
}

describe('the Karvan domain banner', () => {
  it('says the IP opened again, and why', () => {
    const [text, href] = banner({ ...base, access: 'fallback', reason: 'expired' });
    expect(text).toBe(
      "The Karvan domain's certificate has ended, so the server's IP opens the panel again. Open the domain",
    );
    expect(href).toBe('/domains/3');
  });

  it('warns when the certificate ends within 14 days and nothing renews it', () => {
    const [text] = banner({ ...base, endsSoon: '2026-11-08T00:00:00Z' });
    expect(text).toMatch(
      /^The Karvan domain's certificate ends on .+ and does not renew by itself\./,
    );
  });

  it('stays away while all is as set', () => {
    expect(banner(base)[0]).toBe('');
    expect(banner({ ...base, domainId: 0, access: 'all' })[0]).toBe('');
  });
});

describe('the Karvan domain row', () => {
  it('waits for a first sign-in through the domain', () => {
    expect(line({ ...base, access: 'waiting' })).toBe('Waits for a first sign-in through it');
  });

  it('says why the IP opened again', () => {
    expect(line({ ...base, access: 'fallback', reason: 'missing' })).toBe(
      "Karvan can't read the Karvan domain's certificate, so the server's IP opens the panel again.",
    );
  });

  it('says nothing more while only the domain opens the panel', () => {
    expect(line(base)).toBe('');
  });
});
