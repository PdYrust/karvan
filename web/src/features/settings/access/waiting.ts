import { newAddressURL, type NewAddress, type PendingAddress } from '../../../lib/api/account';

/** Where the waiting address opens from this page, and what the operator must know first. */
export interface WaitingLink {
  /** The address to open; null when its new path is no longer known here. */
  url: string | null;
  /** Over plain HTTP a new path starts without this session, so it opens at sign-in. */
  signInFirst: boolean;
}

/**
 * The link to a waiting address. `started` is what this page's own start
 * answered; the path is in that answer only (shown once), so a new path
 * started elsewhere or before a reload cannot be linked again. A port-only
 * change keeps this page's path. The link takes the waiting scheme and,
 * in domain mode, the domain's name; the session stays behind on another
 * host or when HTTPS turns into HTTP.
 */
export function waitingLink(
  pending: PendingAddress,
  started: NewAddress | null,
  base: string,
): WaitingLink {
  const here = new URL(base);
  const scheme = pending.tls === 'off' ? 'http:' : 'https:';
  const host = pending.host || here.hostname;
  const moved = host !== here.hostname || (here.protocol === 'https:' && scheme === 'http:');
  const signInFirst = moved || (pending.pathChanged && here.protocol === 'http:');
  const page = signInFirst ? '' : 'settings/access';
  const at = (to: NewAddress) => {
    const url = new URL(newAddressURL(base, to, page));
    url.protocol = scheme;
    url.hostname = host;
    return url.toString();
  };
  if (started && started.port === pending.port && started.expiresAt === pending.expiresAt) {
    return { url: at(started), signInFirst };
  }
  if (pending.pathChanged) return { url: null, signInFirst };
  const path = here.pathname.replace(/^\/+|\/+$/g, '');
  return { url: at({ port: pending.port, path, expiresAt: pending.expiresAt }), signInFirst };
}
