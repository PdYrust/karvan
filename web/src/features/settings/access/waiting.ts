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
 * change keeps this page's path.
 */
export function waitingLink(
  pending: PendingAddress,
  started: NewAddress | null,
  base: string,
): WaitingLink {
  const here = new URL(base);
  const signInFirst = pending.pathChanged && here.protocol === 'http:';
  const page = signInFirst ? '' : 'settings/access';
  if (started && started.port === pending.port && started.expiresAt === pending.expiresAt) {
    return { url: newAddressURL(base, started, page), signInFirst };
  }
  if (pending.pathChanged) return { url: null, signInFirst };
  const path = here.pathname.replace(/^\/+|\/+$/g, '');
  return {
    url: newAddressURL(base, { port: pending.port, path, expiresAt: pending.expiresAt }, page),
    signInFirst,
  };
}
