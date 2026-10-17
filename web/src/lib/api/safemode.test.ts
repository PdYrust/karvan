import { afterEach, describe, expect, it, vi } from 'vitest';
import { getSafeMode, putSafeMode, turnOffConfirm } from './safemode';

afterEach(() => vi.unstubAllGlobals());

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
