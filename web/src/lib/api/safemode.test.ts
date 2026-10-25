import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  backupDatabase,
  deleteBackup,
  getBackups,
  getSafeMode,
  putSafeMode,
  restoreBackup,
  turnOffConfirm,
} from './safemode';

afterEach(() => vi.unstubAllGlobals());

describe('backups', () => {
  it('lists them', async () => {
    const backup = { id: '9f2c4e1ab37d0c55', jobId: 'j1', size: 1843200 };
    const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify({ backups: [backup] }))));
    vi.stubGlobal('fetch', f);
    await expect(getBackups()).resolves.toEqual([backup]);
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/safe-mode\/backups$/);
  });

  it('puts one back with the typed id and reads where its stream starts', async () => {
    const f = vi.fn(() =>
      Promise.resolve(new Response(JSON.stringify({ jobId: 'j1', after: 41 }), { status: 202 })),
    );
    vi.stubGlobal('fetch', f);
    await expect(restoreBackup('9f2c4e1ab37d0c55', '9f2c4e1ab37d0c55')).resolves.toEqual({
      jobId: 'j1',
      after: 41,
    });
    const call = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    expect(call[0]).toMatch(/safe-mode\/backups\/9f2c4e1ab37d0c55\/restore$/);
    expect(call[1].method).toBe('POST');
    expect(JSON.parse(call[1].body as string)).toEqual({ confirm: '9f2c4e1ab37d0c55' });
  });

  it('deletes one', async () => {
    const f = vi.fn(() => Promise.resolve(new Response(null, { status: 204 })));
    vi.stubGlobal('fetch', f);
    await deleteBackup('9f2c4e1ab37d0c55');
    const call = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    expect(call[0]).toMatch(/safe-mode\/backups\/9f2c4e1ab37d0c55$/);
    expect(call[1].method).toBe('DELETE');
  });

  it('names the database from how it was made', () => {
    expect(backupDatabase['vacuum-into']).toBe('SQLite');
    expect(backupDatabase.pg_dump).toBe('PostgreSQL');
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
