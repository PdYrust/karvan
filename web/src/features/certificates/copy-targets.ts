import type { CertCopy, Certificate, CopyTarget } from '../../lib/api/certificates';
import type { Panel, PanelKind } from '../../lib/api/panels';

/** One row of the Copy to a panel picker. */
export interface CopyChoice {
  key: string;
  target: CopyTarget;
  /** The panel's name and kind; none for the Let's Encrypt live folder. */
  panel?: { name: string; kind?: PanelKind };
  /** Karvan copies it there already. */
  copied: boolean;
}

/** A registered Marzban or PasarGuard panel, not a copy of another one's folder. */
export const isRealPanel = (p: Panel) =>
  p.id > 0 && p.kind !== undefined && !p.findings.some((f) => f.code === 'panel_copy');

const sameTarget = (c: CertCopy, t: CopyTarget) =>
  t.target === 'letsencrypt' ? c.target === 'letsencrypt' : c.panel?.id === t.panelId;

/** The Let's Encrypt live folder first, then every real panel by name. */
export function copyChoices(cert: Certificate, panels: readonly Panel[]): CopyChoice[] {
  const le: CopyTarget = { target: 'letsencrypt' };
  const rows: CopyChoice[] = [
    { key: 'letsencrypt', target: le, copied: cert.copies.some((c) => sameTarget(c, le)) },
  ];
  for (const p of panels.filter(isRealPanel).sort((a, b) => a.name.localeCompare(b.name))) {
    const target: CopyTarget = { target: 'panel', panelId: p.id };
    rows.push({
      key: `panel:${String(p.id)}`,
      target,
      panel: { name: p.name, kind: p.kind },
      copied: cert.copies.some((c) => sameTarget(c, target)),
    });
  }
  return rows;
}
