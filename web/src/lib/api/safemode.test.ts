import { afterEach, describe, expect, it, vi } from 'vitest';
import { getBackups, getSafeMode, putSafeMode, turnOffConfirm } from './safemode';

afterEach(() => vi.unstubAllGlobals());

describe('backups', () => {
  it('lists them', async () => {
    const backup = { id: '9f2c4e1ab37d0c55', jobId: 'j1', size: 1843200 };
    const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify({ backups: [backup] }))));
    vi.stubGlobal('fetch', f);
    await expect(getBackups()).resolves.toEqual([backup]);
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/safe-mode\/backups$/);
  });
});

describe('safe mode calls', () => {
  it('always sends enabled, rollback and the confirmation', async () => {
    const f = vi.fn(() =>
      Promise.resolve(new Response(JSON.stringify({ enabled: false, rollback: 'undo' }))),
    );
    vi.stubGlobal('fetch', f);
    await putSafeMode({ enabled: false, rollback: 'undo' }, turnOffConfirm);
    const call = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    expect(call[1].method).toBe('PUT');
    expect(JSON.parse(call[1].body as string)).toEqual({
      enabled: false,
      rollback: 'undo',
      confirm: 'turn-off-safe-mode',
    });
    await putSafeMode({ enabled: true, rollback: 'backup' });
    const again = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    expect(JSON.parse(again[1].body as string)).toMatchObject({ confirm: '' });
    await getSafeMode();
  });
});
