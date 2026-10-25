import type { UserRow } from '../../../lib/api/migrations';

/** A user's expiry as the preview says it: a date, the first connection, or none. */
export type Expiry = { kind: 'date'; date: Date } | { kind: 'onHold' } | { kind: 'never' };

/**
 * An on hold user without an end date has one: its time starts at the first
 * connection, so it does not read as no expiry.
 */
export function expiryOf(row: Pick<UserRow, 'expire' | 'status'>): Expiry {
  if (row.expire) return { kind: 'date', date: new Date(row.expire) };
  return row.status === 'on_hold' ? { kind: 'onHold' } : { kind: 'never' };
}
