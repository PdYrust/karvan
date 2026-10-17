import { afterEach, describe, expect, it, vi } from 'vitest';
import { getPanels, panelKind, panelRelease, scanPanels } from './panels';

afterEach(() => vi.unstubAllGlobals());

function reply(body: unknown) {
  const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify(body), { status: 200 })));
  vi.stubGlobal('fetch', f);
  return f;
}

describe('panels', () => {
  it('lists and scans', async () => {
    const f = reply({ panels: [{ id: 1, name: 'pasarguard' }] });
    await expect(getPanels()).resolves.toMatchObject([{ id: 1 }]);
    await scanPanels();
    const call = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    expect(call[0]).toMatch(/panels\/scan$/);
    expect(call[1].method).toBe('POST');
  });

  it('names known releases only', () => {
    expect(panelRelease('48a6bcb8bba1')).toBe('5.4.1');
    expect(panelRelease('2b231de97dc3')).toBe('0.8.4');
    expect(panelRelease('ffffffffffff')).toBeUndefined();
    expect(panelRelease(undefined)).toBeUndefined();
  });

  it('prefers the database over the compose hint', () => {
    expect(panelKind({ kind: 'marzban', hint: 'pasarguard' })).toBe('marzban');
    expect(panelKind({ hint: 'pasarguard' })).toBe('pasarguard');
    expect(panelKind({})).toBeUndefined();
  });
});
