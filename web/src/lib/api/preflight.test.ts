import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from './client';
import { checkReportFiles, fileNameOf, getChanges, getPreview, runJob } from './preflight';

afterEach(() => vi.unstubAllGlobals());

function reply(body: BodyInit, init: ResponseInit = { status: 200 }) {
  const f = vi.fn(() => Promise.resolve(new Response(body, init)));
  vi.stubGlobal('fetch', f);
  return f;
}
const urlOf = (f: ReturnType<typeof reply>) => (f.mock.calls.at(-1) as unknown as [string])[0];

describe('getPreview', () => {
  it('asks for one filter, a search and a page of 50', async () => {
    const f = reply(JSON.stringify({ users: [], total: 0 }));
    await getPreview('ab', { group: 'renamed', q: 'ali', offset: 50 });
    expect(urlOf(f)).toMatch(/migrations\/ab\/preview\?group=renamed&q=ali&cursor=50&limit=50$/);
  });
});

describe('getChanges', () => {
  it('leaves the cursor out on the first page', async () => {
    const f = reply(JSON.stringify({ changed: false, users: [], total: 0 }));
    await getChanges('ab');
    expect(urlOf(f)).toMatch(/migrations\/ab\/changes\?limit=50$/);
  });
});

describe('runJob', () => {
  it('sends the plan version and the reload, and no password unless one is given', async () => {
    const f = reply(JSON.stringify({ id: 'ab', state: 'running' }), { status: 202 });
    await runJob('ab', { planVersion: 3, reload: 'touch' });
    const init = (f.mock.calls.at(-1) as unknown as [string, RequestInit])[1];
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body as string)).toEqual({ planVersion: 3, reload: 'touch' });
  });
});

describe('fileNameOf', () => {
  it('takes the name the server gives', () => {
    expect(fileNameOf('attachment; filename="karvan-check-ab.csv"', 'x')).toBe(
      'karvan-check-ab.csv',
    );
  });
  it('drops path parts and falls back when there is no name', () => {
    expect(fileNameOf('attachment; filename="../a/b.csv"', 'x')).toBe('..ab.csv');
    expect(fileNameOf(null, 'karvan-check-ab.json')).toBe('karvan-check-ab.json');
  });
});

describe('checkReportFiles', () => {
  it('saves the CSV through a blob under its own name', async () => {
    reply('key\n', {
      status: 200,
      headers: { 'Content-Disposition': 'attachment; filename="karvan-check-ab.csv"' },
    });
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:1') }));
    const clicked: string[] = [];
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      clicked.push(this.download);
    });
    await checkReportFiles.csv('ab');
    expect(clicked).toEqual(['karvan-check-ab.csv']);
    click.mockRestore();
  });
  it('says why a download failed as an API error', async () => {
    reply(JSON.stringify({ error: { code: 'not_checked' } }), { status: 409 });
    await expect(checkReportFiles.json('ab')).rejects.toBeInstanceOf(ApiError);
  });
});
