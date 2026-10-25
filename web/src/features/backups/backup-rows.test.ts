import { describe, expect, it } from 'vitest';
import type { Panel } from '../../lib/api/panels';
import type { Backup } from '../../lib/api/safemode';
import { backupRows, blockedPanels } from './backup-rows';

const backup = (b: Partial<Backup>): Backup => ({
  id: '9f2c4e1ab37d0c55',
  jobId: 'a'.repeat(24),
  dest: 'sqlite:///var/lib/pasarguard/db.sqlite3',
  size: 1843200,
  method: 'vacuum-into',
  createdAt: '2026-10-25T10:00:00Z',
  panelId: 2,
  jobState: 'done',
  sourceKind: 'marzban',
  sourceOrigin: 'https://old.example.com:8000',
  sourceAdmin: 'reseller',
  ...b,
});

const panel = (p: Partial<Panel>): Panel =>
  ({
    id: 2,
    name: 'karvan-test-pasarguard',
    kind: 'pasarguard',
    findings: [],
    ready: true,
    control: { available: true },
    ...p,
  }) as Panel;

describe('backupRows', () => {
  it('reads each backup with its panel and the move of its job', () => {
    const [r] = backupRows([backup({})], [panel({})]);
    expect(r?.restore).toBeNull();
    expect(r?.removable).toBe(true);
    expect(r?.job).toEqual({
      id: 'a'.repeat(24),
      source: { kind: 'marzban', origin: 'https://old.example.com:8000', admin: 'reseller' },
      panel: { id: 2, name: 'karvan-test-pasarguard', kind: 'pasarguard' },
    });
  });

  it('keeps the backup of a running job and says why it cannot go back', () => {
    const [r] = backupRows([backup({ jobState: 'running' })], [panel({})]);
    expect(r?.restore).toBe('running');
    expect(r?.removable).toBe(false);
  });

  it('turns Put back off for a panel that is gone or that Karvan cannot stop', () => {
    const rows = backupRows(
      [backup({ panelId: 9 }), backup({ id: 'b', panelId: 3 }), backup({ id: 'c', panelId: 3 })],
      [panel({ id: 3, control: { available: false, reason: 'docker_denied' } })],
    );
    expect(rows.map((r) => r.restore)).toEqual(['gone', 'control', 'control']);
    expect(rows[0]?.job.panel.kind).toBe('');
    expect(blockedPanels(rows).map((p) => p.id)).toEqual([3]);
  });

  it('leaves the decision to the server while panel control is not probed', () => {
    const [r] = backupRows([backup({})], [panel({ control: undefined })]);
    expect(r?.restore).toBeNull();
  });

  it('draws no move without the kind of the old panel', () => {
    const [r] = backupRows([backup({ sourceKind: '' })], [panel({})]);
    expect(r?.job.panel.kind).toBe('');
  });
});
