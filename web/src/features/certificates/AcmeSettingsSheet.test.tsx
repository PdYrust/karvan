import { Toasty } from '@cloudflare/kumo/components/toast';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { LocaleProvider } from '../../i18n/locale';
import type { KarvanDomain } from '../../lib/api/karvan';
import { AcmeSettingsSheet } from './AcmeSettingsSheet';

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
  vi.unstubAllGlobals();
});

const acme = {
  staging: false,
  email: '',
  keepPort80: false,
  testDirectory: false,
  hasAccount: true,
};

const karvanOn: KarvanDomain = {
  domainId: 3,
  plain: false,
  shared: false,
  only: true,
  access: 'waiting',
};

/** Serves the two settings the sheet reads, or a 404 for the Karvan domain; records each write. */
function serve(karvan: KarvanDomain | null, failKarvan = false) {
  const writes: string[] = [];
  const json = (status: number, body: unknown) =>
    Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, init: RequestInit) => {
      const path = new URL(url).pathname.replace(/^.*\/api\/v1\//, '');
      const sent = init.method === 'PUT' ? (JSON.parse(String(init.body)) as object) : {};
      if (init.method === 'PUT') writes.push(`${path} ${String(init.body)}`);
      if (path === 'settings/karvan-domain') {
        if (!karvan) return json(404, { error: { code: 'not_found' } });
        if (failKarvan && init.method === 'PUT')
          return json(409, { error: { code: 'karvan_shared' } });
        return json(200, { ...karvan, ...sent });
      }
      return json(200, { ...acme, ...sent });
    }),
  );
  return writes;
}

function render() {
  act(() =>
    root.render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <LocaleProvider>
          <Toasty>
            <AcmeSettingsSheet open onOpenChange={vi.fn()} />
          </Toasty>
        </LocaleProvider>
      </QueryClientProvider>,
    ),
  );
}

/** The switch labelled by the row whose title is name. */
const toggle = (name: string) =>
  [...document.querySelectorAll<HTMLElement>('[role="switch"]')].find(
    (s) => document.getElementById(s.getAttribute('aria-labelledby') ?? '')?.textContent === name,
  );

const plain = 'Plain HTTP for a Karvan domain without a certificate';
const only = 'Only open Karvan on its domain';

describe('the domain settings sheet', () => {
  it('has the two Karvan switches after port 80, and no Save button', async () => {
    serve(karvanOn);
    render();
    await vi.waitFor(() => expect(toggle(only)).toBeDefined());
    const text = document.body.textContent ?? '';
    expect(text.indexOf('Keep port 80 open for renewals')).toBeLessThan(text.indexOf(plain));
    expect(text.indexOf(plain)).toBeLessThan(text.indexOf(only));
    expect(toggle(plain)?.getAttribute('aria-checked')).toBe('false');
    expect(toggle(only)?.getAttribute('aria-checked')).toBe('true');
    expect(text).toContain('karvan access open-ip');
    const buttons = [...document.querySelectorAll('button')].map((b) => b.textContent);
    expect(buttons).not.toContain('Save');
    expect(buttons).not.toContain('Cancel');
  });

  it('saves each Karvan switch as it turns, with no toast', async () => {
    const writes = serve(karvanOn);
    render();
    await vi.waitFor(() => expect(toggle(plain)).toBeDefined());
    await act(async () => {
      toggle(plain)?.click();
      await vi.waitFor(() => expect(writes).toEqual(['settings/karvan-domain {"plain":true}']));
    });
    await vi.waitFor(() => expect(toggle(plain)?.getAttribute('aria-checked')).toBe('true'));
    await act(async () => {
      toggle(only)?.click();
      await vi.waitFor(() => expect(writes).toHaveLength(2));
    });
    expect(writes[1]).toBe('settings/karvan-domain {"only":false}');
    await vi.waitFor(() => expect(toggle(only)?.getAttribute('aria-checked')).toBe('false'));
    expect(document.querySelector('[role="status"], [data-sonner-toast]')).toBeNull();
  });

  it("saves a Let's Encrypt switch at once, and the email when the field is left", async () => {
    const writes = serve(karvanOn);
    render();
    await vi.waitFor(() => expect(toggle('Keep port 80 open for renewals')).toBeDefined());
    await act(async () => {
      toggle('Keep port 80 open for renewals')?.click();
      await vi.waitFor(() => expect(writes).toHaveLength(1));
    });
    expect(writes[0]).toBe('settings/acme {"staging":false,"keepPort80":true,"email":""}');
    const email = document.querySelector<HTMLInputElement>('#acme-email');
    await act(async () => {
      const set = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set;
      set?.call(email, ' ops@example.com ');
      email?.dispatchEvent(new Event('input', { bubbles: true }));
      email?.focus();
      email?.blur();
      await vi.waitFor(() => expect(writes).toHaveLength(2));
    });
    expect(writes[1]).toBe(
      'settings/acme {"staging":false,"keepPort80":true,"email":"ops@example.com"}',
    );
  });

  it('puts a switch back and says why when its save fails', async () => {
    serve(karvanOn, true);
    render();
    await vi.waitFor(() => expect(toggle(plain)).toBeDefined());
    await act(async () => {
      toggle(plain)?.click();
      await vi.waitFor(() =>
        expect(document.body.textContent).toContain("Couldn't save the settings."),
      );
    });
    expect(toggle(plain)?.getAttribute('aria-checked')).toBe('false');
  });

  it('leaves the Karvan switches out when the server cannot change the Karvan domain', async () => {
    serve(null);
    render();
    await vi.waitFor(() => expect(toggle('Keep port 80 open for renewals')).toBeDefined());
    expect(toggle(plain)).toBeUndefined();
    expect(toggle(only)).toBeUndefined();
  });
});
