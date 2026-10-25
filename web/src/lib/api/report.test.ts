import { afterEach, describe, expect, it, vi } from 'vitest';
import { reloadOutcome, reportFiles, restartCommand, rollbackJob } from './report';

afterEach(() => vi.unstubAllGlobals());

function reply(body: BodyInit, init: ResponseInit = { status: 200 }) {
  const f = vi.fn(() => Promise.resolve(new Response(body, init)));
  vi.stubGlobal('fetch', f);
  return f;
}
const callOf = (f: ReturnType<typeof reply>) =>
  f.mock.calls.at(-1) as unknown as [string, RequestInit];

describe('reloadOutcome', () => {
  it('says nothing when the run ended before its reload', () => {
    expect(reloadOutcome({ method: '', ok: 0 }, 'pasarguard')).toEqual({ kind: 'none' });
    expect(reloadOutcome(undefined, 'pasarguard')).toEqual({ kind: 'none' });
  });

  it("gives a manual reload the panel's restart command", () => {
    expect(
      reloadOutcome({ method: 'manual', ok: 0, command: 'marzban restart' }, 'marzban'),
    ).toEqual({ kind: 'manual', command: 'marzban restart' });
  });

  it('never passes on the server text of a failed reload, only the command', () => {
    expect(
      reloadOutcome({ method: 'touch', ok: 0, error: 'dial tcp: refused' }, 'pasarguard'),
    ).toEqual({ kind: 'error', command: 'pasarguard restart' });
  });

  it('counts the users the panel did not pick up, with the "N more" at the end', () => {
    const r = { method: 'touch', ok: 1160, failed: ['ali: 500', 'sara: timeout', '8 more'] };
    expect(reloadOutcome(r, 'pasarguard')).toEqual({
      kind: 'partial',
      ok: 1160,
      failed: 10,
      command: 'pasarguard restart',
    });
  });

  it('reads a reload every user took as done', () => {
    expect(reloadOutcome({ method: 'core_restart', ok: 9 }, 'marzban')).toEqual({
      kind: 'ok',
      method: 'core_restart',
      ok: 9,
    });
  });

  it('names each panel kind its own restart', () => {
    expect(restartCommand('marzban')).toBe('marzban restart');
    expect(restartCommand('pasarguard')).toBe('pasarguard restart');
  });
});

describe('rollbackJob', () => {
  it('posts with the CSRF header and no body', async () => {
    const f = reply(JSON.stringify({ id: 'ab' }), { status: 202 });
    await expect(rollbackJob('ab')).resolves.toEqual({ id: 'ab' });
    const [url, init] = callOf(f);
    expect(url).toMatch(/migrations\/ab\/rollback$/);
    expect(init.method).toBe('POST');
    expect(init.body).toBeNull();
    expect(new Headers(init.headers).has('X-CSRF-Token')).toBe(true);
  });
});

describe('reportFiles', () => {
  it('asks for the admin password file with a POST, never a GET', async () => {
    const f = reply('admin,password\n', {
      status: 200,
      headers: { 'Content-Disposition': 'attachment; filename="karvan-passwords-ab.csv"' },
    });
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:1') }));
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined);
    await reportFiles.passwords('ab');
    const [url, init] = callOf(f);
    expect(url).toMatch(/migrations\/ab\/passwords\.csv$/);
    expect(init.method).toBe('POST');
    expect(click).toHaveBeenCalledOnce();
    click.mockRestore();
  });

  it('reads the report and the new links with a GET', async () => {
    const f = reply('{}', { status: 200 });
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:1') }));
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined);
    await reportFiles.links('ab');
    expect(callOf(f)[0]).toMatch(/migrations\/ab\/links\.csv$/);
    expect(callOf(f)[1].method).toBe('GET');
    await reportFiles.json('ab');
    expect(callOf(f)[0]).toMatch(/migrations\/ab\/report$/);
    click.mockRestore();
  });
});
