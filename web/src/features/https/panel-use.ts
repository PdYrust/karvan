import type { Domain } from '../../lib/api/domains';
import type { Https } from '../../lib/api/https';

/** What a domain's menu offers for the Karvan panel, and why Use cannot be picked yet. */
export type PanelUse =
  { kind: 'use'; blocked?: 'no_certificate' | 'domain_mismatch' } | { kind: 'stop' };

/**
 * Stop while the panel is served with this domain's certificate, else Use: picked only
 * when the domain has a certificate that covers its name.
 */
export function panelUseOf(d: Domain, h: Https): PanelUse {
  if (h.tls === 'domain' && h.domainId === d.id) return { kind: 'stop' };
  if (!d.certificate) return { kind: 'use', blocked: 'no_certificate' };
  if (d.warnings.includes('domain_mismatch')) return { kind: 'use', blocked: 'domain_mismatch' };
  return { kind: 'use' };
}

/** The HTTPS tab with its change dialog open for Use or Stop on domain d. */
export const panelLink = (d: { id: number }, kind: PanelUse['kind']) =>
  `/settings/https?domain=${String(d.id)}&panel=${kind}`;
