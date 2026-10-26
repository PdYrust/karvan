import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, useEffect } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { MemoryRouter, useLocation } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { panelsKey } from '../../features/panels/use-panels';
import { LocaleProvider } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { SidebarNav } from './SidebarNav';

// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

const panels = [{ id: 1, name: 'marzban-a', kind: 'marzban', ready: true }] as Panel[];

let host: HTMLDivElement;
let root: Root;
let path = '';

function Where({ onPath }: { onPath: (p: string) => void }) {
  const { pathname } = useLocation();
  useEffect(() => {
    onPath(pathname);
  });
  return null;
}

function render(at: string, onNavigate: () => void) {
  const client = new QueryClient();
  client.setQueryData(panelsKey, panels);
  act(() =>
    root.render(
      <QueryClientProvider client={client}>
        <LocaleProvider>
          <MemoryRouter initialEntries={[at]}>
            <SidebarNav onNavigate={onNavigate} />
            <Where
              onPath={(p) => {
                path = p;
              }}
            />
          </MemoryRouter>
        </LocaleProvider>
      </QueryClientProvider>,
    ),
  );
}

const group = () => host.querySelector<HTMLButtonElement>('button[aria-expanded]');
const subLink = () => host.querySelector<HTMLAnchorElement>('a[href="/analytics/server"]');

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

describe('SidebarNav', () => {
  it('opens and closes a row with sub-pages in place, without navigating', () => {
    const onNavigate = vi.fn();
    render('/panels', onNavigate);
    expect(group()?.getAttribute('aria-expanded')).toBe('false');
    expect(subLink()).toBeNull();
    act(() => group()?.click());
    expect(group()?.getAttribute('aria-expanded')).toBe('true');
    expect(subLink()).toBeTruthy();
    expect(path).toBe('/panels');
    expect(onNavigate).not.toHaveBeenCalled();
    act(() => group()?.click());
    expect(subLink()).toBeNull();
  });

  it('starts open on one of its pages, and the reader can close it', () => {
    render('/analytics/server', vi.fn());
    expect(group()?.getAttribute('aria-expanded')).toBe('true');
    act(() => group()?.click());
    expect(group()?.getAttribute('aria-expanded')).toBe('false');
  });

  it('navigates from a sub-row and tells the sheet to close', () => {
    const onNavigate = vi.fn();
    render('/panels', onNavigate);
    act(() => group()?.click());
    act(() => subLink()?.click());
    expect(path).toBe('/analytics/server');
    expect(onNavigate).toHaveBeenCalledOnce();
    // Moving into the section keeps it open.
    expect(group()?.getAttribute('aria-expanded')).toBe('true');
  });
});
