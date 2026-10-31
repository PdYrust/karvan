import type { Domain } from '../../lib/api/domains';
import type { Https } from '../../lib/api/https';
import { formOf, type HttpsForm } from './https-form';
import type { PanelUse } from './panel-use';

/** The change a panelLink asks the HTTPS tab to open with, or null when it asks for none. */
export function askedForm(
  query: URLSearchParams,
  domains: readonly Domain[],
  h: Https,
): HttpsForm | null {
  const kind = query.get('panel');
  const d = domains.find((x) => String(x.id) === query.get('domain'));
  return d && (kind === 'use' || kind === 'stop') ? panelForm(d, h, kind) : null;
}

/**
 * The HTTPS change the dialog opens with: this domain's certificate on the same port, or the
 * self-signed one to stop. Leaving a shared port needs a port of its own, and so does the
 * self-signed certificate, whose new address the domain's name would not reach on this port.
 */
export function panelForm(d: Domain, h: Https, kind: PanelUse['kind']): HttpsForm {
  return kind === 'use'
    ? {
        ...formOf(h),
        tls: 'domain',
        domainId: d.id,
        where: h.shared ? 'typed' : 'keep',
        sharedPort: 0,
      }
    : { ...formOf(h), tls: 'self', domainId: 0, where: 'typed', sharedPort: 0 };
}
