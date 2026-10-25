import { describe, expect, it } from 'vitest';
import { expiryOf } from './expiry';

describe('expiryOf', () => {
  it('gives the end date when there is one', () => {
    expect(expiryOf({ expire: '2026-11-24T10:00:00Z', status: 'active' })).toEqual({
      kind: 'date',
      date: new Date('2026-11-24T10:00:00Z'),
    });
    expect(expiryOf({ expire: '2026-11-24T10:00:00Z', status: 'on_hold' }).kind).toBe('date');
  });

  it('says an on hold user without an end starts at the first connection', () => {
    expect(expiryOf({ expire: null, status: 'on_hold' })).toEqual({ kind: 'onHold' });
  });

  it('leaves other users without an end as no expiry', () => {
    expect(expiryOf({ expire: null, status: 'active' })).toEqual({ kind: 'never' });
    expect(expiryOf({ expire: null, status: 'disabled' })).toEqual({ kind: 'never' });
  });
});
