import { afterEach, describe, expect, it, vi } from 'vitest';
import { databaseOf, engineName, getPanels, panelKind, panelRelease, scanPanels } from './panels';

afterEach(() => vi.unstubAllGlobals());

function reply(body: unknown) {
  const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify(body), { status: 200 })));
  vi.stubGlobal('fetch', f);
  return f;
}

function lastCall(f: ReturnType<typeof reply>) {
  return f.mock.calls.at(-1) as unknown as [string, RequestInit];
}

describe('panels', () => {
  it('lists and scans', async () => {
    const f = reply({ panels: [{ id: 1, name: 'pasarguard' }] });
    await expect(getPanels()).resolves.toMatchObject([{ id: 1 }]);
    expect(lastCall(f)[0]).toMatch(/api\/v1\/panels$/);
    await scanPanels();
    expect(lastCall(f)[0]).toMatch(/panels\/scan$/);
    expect(lastCall(f)[1].method).toBe('POST');
  });

  it('asks for a fresh look at panel control only when told to', async () => {
    const f = reply({ panels: [] });
    await getPanels(true);
    expect(lastCall(f)[0]).toMatch(/api\/v1\/panels\?fresh=1$/);
  });

  it('names known releases only', () => {
    expect(panelRelease('48a6bcb8bba1')).toBe('5.4.1');
    expect(panelRelease('c8d2e4f6a1b3')).toBe('5.5.1');
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

describe('engineName', () => {
  it('writes each engine the way it writes itself', () => {
    expect(engineName('sqlite')).toBe('SQLite');
    expect(engineName('mariadb')).toBe('MariaDB');
    expect(engineName('postgres')).toBe('PostgreSQL');
  });

  it('keeps the name of an engine it does not know', () => {
    expect(engineName('cockroach')).toBe('cockroach');
  });
});

describe('databaseOf', () => {
  it('gives a SQLite file as its path', () => {
    expect(databaseOf({ db: 'sqlite:///var/lib/marzban/db.sqlite3', flavor: 'sqlite' })).toEqual({
      engine: 'sqlite',
      where: '/var/lib/marzban/db.sqlite3',
    });
  });

  it('gives a server as its host and port, without the user or the database', () => {
    expect(databaseOf({ db: 'mysql://marzban@127.0.0.1:3306/marzban', flavor: 'mariadb' })).toEqual(
      { engine: 'mariadb', where: '127.0.0.1:3306' },
    );
    expect(databaseOf({ db: 'postgres://pg@::1:5432/pasarguard', flavor: 'postgres' })).toEqual({
      engine: 'postgres',
      where: '::1:5432',
    });
  });

  it('gives a socket as its path', () => {
    expect(
      databaseOf({ db: 'mysql://marzban@unix(/run/mysqld/mysqld.sock)/marzban', flavor: 'mysql' }),
    ).toEqual({ engine: 'mysql', where: '/run/mysqld/mysqld.sock' });
  });

  it('falls back to the address scheme when the server could not be reached', () => {
    expect(databaseOf({ db: 'postgres://pg@db:5432/pasarguard' })).toEqual({
      engine: 'postgres',
      where: 'db:5432',
    });
    expect(databaseOf({ db: 'sqlite:///data/db.sqlite3' }).engine).toBe('sqlite');
  });

  it('knows nothing without an address', () => {
    expect(databaseOf({})).toEqual({ engine: undefined, where: undefined });
    expect(databaseOf({ db: 'nonsense' })).toEqual({ engine: undefined, where: undefined });
  });
});
