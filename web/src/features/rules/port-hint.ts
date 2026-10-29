import type { Ports } from '../../lib/api/ports';

/**
 * The old panel's own port when the rule takes over its domain (D14): the
 * port of the old address when its host is one of the rule's domains.
 */
export function oldPort(sourceUrl: string, domains: readonly string[]): number {
  let u: URL;
  try {
    u = new URL(sourceUrl);
  } catch {
    return 0;
  }
  if (!domains.includes(u.hostname.toLowerCase())) return 0;
  if (u.port) return Number(u.port);
  return u.protocol === 'https:' ? 443 : 0;
}

/** What to say under the port field: nothing, a free port for an empty field, or a free one for a taken one. */
export type PortHint =
  { kind: 'free'; free: number } | { kind: 'taken'; port: number; free: number };

/**
 * The proposal shown under the port (D14): only when the field is empty
 * or its port is held by something other than a rule, since rules share
 * a port by name.
 */
export function portHint(ports: Ports | undefined, typed: string): PortHint | null {
  if (!ports || ports.proposal < 1) return null;
  if (typed === '') return { kind: 'free', free: ports.proposal };
  const port = Number(typed);
  const owner = ports.ports.find((p) => p.port === port)?.owner;
  if (!owner || owner === 'rule' || port === ports.proposal) return null;
  return { kind: 'taken', port, free: ports.proposal };
}
