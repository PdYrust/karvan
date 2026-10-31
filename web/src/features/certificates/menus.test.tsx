import { Toasty } from '@cloudflare/kumo/components/toast';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, useEffect, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { MemoryRouter, useLocation } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { LocaleProvider } from '../../i18n/locale';
import type { Certificate } from '../../lib/api/certificates';
import type { Domain } from '../../lib/api/domains';
import type { Panel } from '../../lib/api/panels';
import { panelsKey } from '../panels/use-panels';
import { CertMenu } from './CertMenu';
import { DomainMenu } from './DomainMenu';

// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement;
let root: Root;
let at = '';

function Where({ onPath }: { onPath: (p: string) => void }) {
  const { pathname, search } = useLocation();
  useEffect(() => {
    onPath(pathname + search);
  });
  return null;
}

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

const facts = { domains: ['panel.example.com'], issuer: 'R11', warnings: [] };
const domain: Domain = {
  id: 3,
  name: 'panel.example.com',
  certificate: { id: 5, name: 'panel', origin: '', facts },
  certificates: [],
  pins: [],
  acme: null,
  rules: [],
  warnings: [],
};
const cert = {
  id: 5,
  name: 'panel',
  kind: 'pasted',
  origin: '',
  hasKey: true,
  facts,
  domains: [{ id: 3, name: 'panel.example.com', warnings: [] }],
  rules: [],
  found: false,
  copies: [],
} as Certificate;
const panels = [
  { id: 1, name: 'marzban', kind: 'marzban', findings: [] },
  { id: 2, name: 'pasarguard', kind: 'pasarguard', findings: [] },
] as unknown as Panel[];

function render(node: ReactNode) {
  const client = new QueryClient();
  client.setQueryData(panelsKey, panels);
  act(() =>
    root.render(
      <QueryClientProvider client={client}>
        <LocaleProvider>
          <Toasty>
            <MemoryRouter>
              {node}
              <Where
                onPath={(p) => {
                  at = p;
                }}
              />
            </MemoryRouter>
          </Toasty>
        </LocaleProvider>
      </QueryClientProvider>,
    ),
  );
}

/** Opens the menu of the "..." button and returns its items' words. */
async function open(label: string): Promise<string[]> {
  const trigger = host.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
  expect(trigger).not.toBeNull();
  await act(async () => {
    trigger?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, button: 0 }));
    trigger?.click();
    await Promise.resolve();
  });
  return [...document.querySelectorAll('[role="menuitem"]')].map((m) => m.textContent ?? '');
}

describe('the certificate menu', () => {
  it('has Copy to a panel and Delete, and no Edit', async () => {
    render(<CertMenu cert={cert} onDelete={vi.fn()} />);
    expect(await open('More for panel')).toEqual(['Copy to a panel', 'Delete']);
  });

  it("opens the picker with Let's Encrypt live and every panel", async () => {
    render(<CertMenu cert={cert} onDelete={vi.fn()} />);
    await open('More for panel');
    const sub = [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
      (m) => m.textContent === 'Copy to a panel',
    );
    await act(async () => {
      sub?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, button: 0 }));
      sub?.click();
      await Promise.resolve();
    });
    const items = [...document.querySelectorAll('[role="menuitem"]')].map((m) => m.textContent);
    expect(items).toEqual(
      expect.arrayContaining(["Let's Encrypt live folder", 'marzban', 'pasarguard']),
    );
  });
});

/** The menu item with these words. */
const item = (words: string) =>
  [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
    (m) => m.textContent === words,
  );

/** Answers PUT settings/karvan-domain with the domain it was sent, and records the bodies. */
function serveKarvan(): unknown[] {
  const sent: unknown[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn((_url: string, init: RequestInit) => {
      const body = JSON.parse(String(init.body)) as { domainId: number };
      sent.push(body);
      const out = { domainId: body.domainId, plain: false, shared: false };
      return Promise.resolve(
        new Response(JSON.stringify(out), { headers: { 'Content-Type': 'application/json' } }),
      );
    }),
  );
  return sent;
}

const none = { domainId: 0, plain: false, shared: false };

describe('the domain menu', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('offers Use for Karvan, which enables the domain at once', async () => {
    const sent = serveKarvan();
    render(<DomainMenu domain={domain} onDelete={vi.fn()} karvan={none} />);
    const items = await open('More for panel.example.com');
    expect(items).toEqual(['Change certificate', 'Use for Karvan', 'Delete']);
    await act(async () => {
      item('Use for Karvan')?.click();
      await vi.waitFor(() => expect(sent).toEqual([{ domainId: 3 }]));
    });
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain('Karvan uses panel.example.com now'),
    );
    expect(at).toBe('/');
  });

  it('says Stop while Karvan uses the domain, which stops it, and keeps Delete off', async () => {
    const sent = serveKarvan();
    render(<DomainMenu domain={domain} onDelete={vi.fn()} karvan={{ ...none, domainId: 3 }} />);
    const items = await open('More for panel.example.com');
    expect(items).toEqual(['Change certificate', 'Stop using for Karvan', 'Delete']);
    expect(item('Delete')?.getAttribute('aria-disabled')).toBe('true');
    await act(async () => {
      item('Stop using for Karvan')?.click();
      await vi.waitFor(() => expect(sent).toEqual([{ domainId: 0 }]));
    });
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain('Karvan stopped using panel.example.com'),
    );
  });

  it('can enable a domain without a certificate, and leaves Karvan out for an IP or when unknown', async () => {
    render(
      <DomainMenu domain={{ ...domain, certificate: null }} onDelete={vi.fn()} karvan={none} />,
    );
    expect(await open('More for panel.example.com')).toContain('Use for Karvan');
    expect(item('Use for Karvan')?.getAttribute('aria-disabled')).toBeNull();
    act(() => root.render(<span />));
    render(
      <DomainMenu domain={{ ...domain, name: '203.0.113.9' }} onDelete={vi.fn()} karvan={none} />,
    );
    expect(await open('More for 203.0.113.9')).toEqual(['Change certificate', 'Delete']);
    act(() => root.render(<span />));
    render(<DomainMenu domain={domain} onDelete={vi.fn()} />);
    expect(await open('More for panel.example.com')).toEqual(['Change certificate', 'Delete']);
  });
});
