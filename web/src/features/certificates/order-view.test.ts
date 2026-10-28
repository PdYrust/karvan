import { describe, expect, it } from 'vitest';
import type { AcmeOrder, AcmeState } from '../../lib/api/acme';
import { orderPoll, orderSteps, retryTime } from './order-view';

const order = (state: AcmeState, patch: Partial<AcmeOrder> = {}): AcmeOrder => ({
  id: 7,
  method: 'http-01',
  renewal: false,
  state,
  dns: null,
  error: null,
  createdAt: '2026-10-28T12:00:00Z',
  updatedAt: '2026-10-28T12:00:02Z',
  expiresAt: null,
  ...patch,
});

describe('the order steps', () => {
  it('mark the steps before the current one done', () => {
    expect(orderSteps(order('validating'))).toEqual([
      { id: 'start', state: 'done' },
      { id: 'check', state: 'current' },
      { id: 'issue', state: 'todo' },
      { id: 'done', state: 'todo' },
    ]);
  });
  it('put a DNS order waiting for its record on the check step', () => {
    expect(orderSteps(order('waiting_dns'))[1]).toEqual({ id: 'check', state: 'current' });
  });
  it('start on the first step and end with every step done', () => {
    expect(orderSteps(order('starting'))[0]).toEqual({ id: 'start', state: 'current' });
    expect(orderSteps(order('done')).every((s) => s.state === 'done')).toBe(true);
  });
  it('show no steps for an order that failed or was cancelled', () => {
    expect(orderSteps(order('failed'))).toEqual([]);
    expect(orderSteps(order('cancelled'))).toEqual([]);
  });
});

describe('reading the order again', () => {
  it('happens every 2 seconds while it runs', () => {
    expect(orderPoll(order('issuing'))).toBe(2000);
    expect(orderPoll(order('waiting_dns'))).toBe(2000);
  });
  it('stops once it ends or there is none', () => {
    expect(orderPoll(order('done'))).toBe(false);
    expect(orderPoll(order('failed'))).toBe(false);
    expect(orderPoll(null)).toBe(false);
  });
});

describe('a rate limit', () => {
  it('gives the time to try again', () => {
    const o = order('failed', {
      error: { code: 'acme_rate_limited', detail: '', retryAt: '2026-10-29T08:00:00Z' },
    });
    expect(retryTime(o.error)).toEqual(new Date('2026-10-29T08:00:00Z'));
  });
  it('gives none without one', () => {
    expect(retryTime({ code: 'acme_failed', detail: '' })).toBeUndefined();
    expect(retryTime(null)).toBeUndefined();
  });
});
