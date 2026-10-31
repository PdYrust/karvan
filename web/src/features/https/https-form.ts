import type { Domain } from '../../lib/api/domains';
import type { Https, HttpsStart, PendingHttps, TlsMode } from '../../lib/api/https';
import { latinDigits } from '../../lib/format/number';

/** Where the new address answers: this port, a typed one, or a bridge HTTPS port. */
export type Where = 'keep' | 'typed' | 'shared';

export interface HttpsForm {
  tls: TlsMode;
  domainId: number;
  where: Where;
  typed: string;
  sharedPort: number;
}

export function formOf(h: Https): HttpsForm {
  return {
    tls: h.tls,
    domainId: h.domainId,
    where: h.shared ? 'shared' : 'keep',
    typed: '',
    sharedPort: h.shared ? h.port : 0,
  };
}

/** A typed port: at most five digits, Persian ones read as the same digits. */
export const typedDigits = (typed: string) => latinDigits(typed).replace(/\D/g, '').slice(0, 5);

/** Domains the panel can be served for: those with a certificate that covers their name. */
export function certifiedDomains(domains: readonly Domain[]): Domain[] {
  return domains.filter((d) => d.certificate && !d.warnings.includes('domain_mismatch'));
}

/** The bridge ports where an HTTPS rule holds the domain. */
export function sharedPortsOf(h: Https, domain: Domain | undefined): number[] {
  return domain ? h.sharedPorts.filter((s) => s.domain === domain.name).map((s) => s.port) : [];
}

/** The change to send, or null while the form cannot be sent or changes nothing. */
export function httpsStart(f: HttpsForm, h: Https): HttpsStart | null {
  const domain = f.tls === 'domain';
  if (domain && f.domainId < 1) return null;
  const shared = domain && f.where === 'shared';
  let port = 0;
  if (shared) {
    if (f.sharedPort < 1) return null;
    port = f.sharedPort;
  } else if (f.where === 'typed') {
    if (!/^\d{1,5}$/.test(f.typed)) return null;
    port = Number(f.typed);
    if (port < 1 || port > 65535) return null;
  }
  // Leaving a shared port needs a port of its own.
  if (h.shared && !shared && port === 0) return null;
  const domainId = domain ? f.domainId : 0;
  const samePort = port === 0 || port === h.port;
  if (f.tls === h.tls && domainId === (domain ? h.domainId : 0) && shared === h.shared && samePort)
    return null;
  return { tls: f.tls, domainId, port: samePort && !shared ? 0 : port, shared };
}

/** Where a waiting change opens from this page, and whether this session carries over. */
export interface HttpsLink {
  /** Null when a new hidden path waits too: only the page that started it knows the path. */
  url: string | null;
  signInFirst: boolean;
}

/**
 * The address a waiting change serves, at the HTTPS tab: its scheme, the
 * domain's name in domain mode, its port, and this page's path. The
 * session goes along on the same host unless HTTPS turns into HTTP, where
 * the browser keeps the secure cookie back.
 */
export function httpsLink(p: PendingHttps, domains: readonly Domain[], base: string): HttpsLink {
  const here = new URL(base);
  const url = new URL(base);
  url.protocol = p.tls === 'off' ? 'http:' : 'https:';
  const name = p.tls === 'domain' ? domains.find((d) => d.id === p.domainId)?.name : undefined;
  if (name && !name.startsWith('*.')) url.hostname = name;
  url.port = String(p.port);
  url.pathname = `${here.pathname.replace(/\/*$/, '/')}settings/https`;
  url.search = '';
  url.hash = '';
  const signInFirst =
    url.hostname !== here.hostname || (here.protocol === 'https:' && url.protocol === 'http:');
  return { url: p.pathChanged ? null : url.toString(), signInFirst };
}
