import { Toasty } from '@cloudflare/kumo/components/toast';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { LocaleProvider } from '../../i18n/locale';
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

/** Serves the two settings the sheet reads, or a 404 for the Karvan domain; records each write. */
function serve(karvan: { domainId: number; plain: boolean; shared: boolean } | null) {
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
      if (init.method === 'PUT') writes.push(`${path} ${String(init.body)}`);
      if (path === 'settings/karvan-domain') {
        if (!karvan) return json(404, { error: { code: 'not_found' } });
        const sent = init.method === 'PUT' ? (JSON.parse(String(init.body)) as object) : {};
        return json(200, { ...karvan, ...sent });
      }
      return json(200, acme);
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

const radio = (name: string) =>
  [...document.querySelectorAll<HTMLElement>('[role="radio"]')].find((r) =>
    r.closest('label')?.textContent?.startsWith(name),
  );

const button = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>('button')].find((b) => b.textContent === name);

describe('the domain settings sheet', () => {
  it('has the choice for the Karvan domain without a certificate next to port 80', async () => {
    serve({ domainId: 3, plain: false, shared: false });
    render();
    await vi.waitFor(() => expect(radio('Plain HTTP')).toBeDefined());
    const text = document.body.textContent ?? '';
    expect(text.indexOf('Keep port 80 open for renewals')).toBeLessThan(
      text.indexOf('For the Karvan domain without a certificate'),
    );
    expect(radio("Karvan's own certificate")?.getAttribute('aria-checked')).toBe('true');
    expect(radio('Plain HTTP')?.getAttribute('aria-checked')).toBe('false');
  });

  it("saves plain HTTP for the Karvan domain, then the Let's Encrypt settings", async () => {
    const writes = serve({ domainId: 3, plain: false, shared: false });
    render();
    await vi.waitFor(() => expect(radio('Plain HTTP')).toBeDefined());
    act(() => radio('Plain HTTP')?.click());
    expect(radio('Plain HTTP')?.getAttribute('aria-checked')).toBe('true');
    await act(async () => {
      button('Save')?.click();
      await vi.waitFor(() => expect(writes).toHaveLength(2));
    });
    expect(writes[0]).toBe('settings/karvan-domain {"plain":true}');
    expect(writes[1]).toMatch(/^settings\/acme /);
  });

  it("saves only the Let's Encrypt settings when the choice stays", async () => {
    const writes = serve({ domainId: 0, plain: true, shared: false });
    render();
    await vi.waitFor(() => expect(radio('Plain HTTP')?.getAttribute('aria-checked')).toBe('true'));
    await act(async () => {
      button('Save')?.click();
      await vi.waitFor(() => expect(writes).toHaveLength(1));
    });
    expect(writes[0]).toMatch(/^settings\/acme /);
  });

  it('leaves the choice out when the server cannot change the Karvan domain', async () => {
    serve(null);
    render();
    await vi.waitFor(() => expect(button('Save')).toBeDefined());
    expect(document.body.textContent).not.toContain('For the Karvan domain without a certificate');
  });
});
