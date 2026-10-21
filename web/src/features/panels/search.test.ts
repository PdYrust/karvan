import { describe, expect, it } from 'vitest';
import type { Panel } from '../../lib/api/panels';
import { matchesPanel } from './search';

const pasarguard: Panel = {
  id: 2,
  name: 'karvan-test-pasarguard',
  dir: '/opt/pasarguard',
  envPath: '/opt/pasarguard/.env',
  kind: 'pasarguard',
  revision: '48a6bcb8bba1',
  db: 'postgres://pasarguard@127.0.0.1:5432/pasarguard',
  flavor: 'postgres',
  settings: { subPath: 'sub', host: '0.0.0.0', port: 8000 },
  findings: [],
  ready: true,
};

describe('matchesPanel', () => {
  it('matches everything for an empty query', () => {
    expect(matchesPanel(pasarguard, '')).toBe(true);
    expect(matchesPanel(pasarguard, '   ')).toBe(true);
  });

  it('finds a panel by its name, folder, database and release, in any case', () => {
    expect(matchesPanel(pasarguard, 'TEST-PASAR')).toBe(true);
    expect(matchesPanel(pasarguard, '/opt/pas')).toBe(true);
    expect(matchesPanel(pasarguard, '127.0.0.1:5432')).toBe(true);
    expect(matchesPanel(pasarguard, 'postgresql')).toBe(true);
    expect(matchesPanel(pasarguard, '5.4.1')).toBe(true);
  });

  it('finds a panel by its kind in either language', () => {
    expect(matchesPanel(pasarguard, 'PasarGuard')).toBe(true);
    expect(matchesPanel(pasarguard, 'پاسارگارد')).toBe(true);
    expect(matchesPanel(pasarguard, 'مرزبان')).toBe(false);
    expect(matchesPanel({ ...pasarguard, kind: undefined, hint: 'marzban' }, 'marzban')).toBe(true);
  });

  it('leaves out a panel that does not match', () => {
    expect(matchesPanel(pasarguard, 'mysql')).toBe(false);
  });
});
