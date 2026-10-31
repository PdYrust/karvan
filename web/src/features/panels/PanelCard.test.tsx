import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { LocaleProvider } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { PanelCard } from './PanelCard';

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

const base = {
  id: 2,
  name: 'pasarguard',
  kind: 'pasarguard',
  revision: '48a6bcb8bba1',
  db: 'sqlite:////var/lib/pasarguard/db.sqlite3',
  findings: [],
  ready: true,
} as unknown as Panel;

/** The card's three lines as text: the top row, the quiet line, the status line. */
function card(over: Partial<Panel>, current = false): string[] {
  const panel = { ...base, ...over };
  act(() =>
    root.render(
      <LocaleProvider>
        <MemoryRouter>
          <PanelCard panel={panel} current={current} menu={<button type="button">menu</button>} />
        </MemoryRouter>
      </LocaleProvider>,
    ),
  );
  const article = host.querySelector('article');
  const top = article?.firstElementChild;
  const lines = [...host.querySelectorAll('article p')];
  return [top?.textContent ?? '', ...lines.map((p) => p.textContent ?? '')];
}

describe('PanelCard', () => {
  it('shows the name and menu, the version and database, then only Ready', () => {
    const [top, quiet, status] = card({ control: { available: true } });
    expect(top).toBe('pasarguardmenu');
    expect(quiet).toBe('PasarGuard 5.4.1SQLite');
    expect(status).toBe('Ready');
    expect(host.querySelector('a')?.getAttribute('href')).toBe('/panels/2');
  });

  it('leaves the database address to the panel page', () => {
    card({ control: { available: true } });
    expect(host.textContent).not.toContain('/var/lib/pasarguard');
    expect(host.querySelectorAll('article p')).toHaveLength(2);
  });

  it('adds one warning to the status line when panel control is missing', () => {
    const [, , status] = card({ control: { available: false, reason: 'docker_denied' } });
    expect(status).toBe('ReadyPanel control: No Docker access');
  });

  it('keeps the not-ready state', () => {
    const [, , status] = card({
      ready: false,
      findings: [{ code: 'schema_mismatch' }] as Panel['findings'],
      control: { available: true },
    });
    expect(status).toBe('1 problem');
  });

  it('cuts a long name on its line and keeps the tag and the menu', () => {
    const name = 'pasarguard-production-europe-frankfurt-second-server';
    const [top] = card({ name }, true);
    expect(top).toBe(`${name}Currentmenu`);
    const link = host.querySelector('a');
    expect(link?.className).toContain('truncate');
    expect(link?.className).toContain('min-w-0');
    // A Latin name in Persian is cut at its end.
    expect(link?.getAttribute('dir')).toBe('auto');
  });
});
