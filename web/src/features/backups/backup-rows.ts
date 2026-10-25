import type { JobRow } from '../../lib/api/migrations';
import { panelKind, type Panel } from '../../lib/api/panels';
import type { Backup } from '../../lib/api/safemode';

/** Why Put back is off for a backup: its job runs, its panel is gone, or Karvan can't stop the panel. */
export type RestoreBlock = 'running' | 'gone' | 'control';

export interface BackupRow {
  backup: Backup;
  panel: Panel | undefined;
  /** The job that took it, as the job labels read one. */
  job: Pick<JobRow, 'id' | 'source' | 'panel'>;
  restore: RestoreBlock | null;
  /** A running job's backup stays. */
  removable: boolean;
}

function restoreBlock(b: Backup, panel: Panel | undefined): RestoreBlock | null {
  if (b.jobState === 'running') return 'running';
  if (!panel) return 'gone';
  // Without a probe yet the server decides, and says why if it can't.
  if (panel.control && !panel.control.available) return 'control';
  return null;
}

/** Each backup with its panel, its job and what can be done with it. */
export function backupRows(backups: readonly Backup[], panels: readonly Panel[]): BackupRow[] {
  const byId = new Map(panels.map((p) => [p.id, p]));
  return backups.map((b) => {
    const panel = b.panelId > 0 ? byId.get(b.panelId) : undefined;
    const kind = panel ? panelKind(panel) : undefined;
    return {
      backup: b,
      panel,
      job: {
        id: b.jobId,
        source: { kind: b.sourceKind || 'marzban', origin: b.sourceOrigin, admin: b.sourceAdmin },
        // Without both kinds no move is drawn, rather than a wrong logo.
        panel: { id: b.panelId, name: panel?.name ?? '', kind: b.sourceKind ? (kind ?? '') : '' },
      },
      restore: restoreBlock(b, panel),
      removable: b.jobState !== 'running',
    };
  });
}

/** The panels Karvan can't stop and start that hold backups, each once, in list order. */
export function blockedPanels(rows: readonly BackupRow[]): Panel[] {
  const seen = new Map<number, Panel>();
  for (const r of rows) {
    if (r.restore === 'control' && r.panel && !seen.has(r.panel.id)) seen.set(r.panel.id, r.panel);
  }
  return [...seen.values()];
}
