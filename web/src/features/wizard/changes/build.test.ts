import { describe, expect, it } from 'vitest';
import { buildOp, buildScope, emptyOpForm, emptyScopeForm, type OpForm } from './build';

const form = (over: Partial<OpForm>): OpForm => ({ ...emptyOpForm, ...over });

describe('buildOp', () => {
  it('reads signed traffic and days', () => {
    expect(buildOp('traffic.add', form({ amount: '+10' }), 'UTC')).toEqual({
      kind: 'traffic.add',
      params: { gb: '10' },
    });
    expect(buildOp('time.add', form({ amount: '-5' }), 'UTC')).toEqual({
      kind: 'time.add',
      params: { days: '-5' },
    });
    expect(buildOp('time.add', form({ amount: '1.5' }), 'UTC')).toBeNull();
  });

  it('takes only positive limits and periods', () => {
    expect(buildOp('traffic.set', form({ amount: '-5' }), 'UTC')).toBeNull();
    expect(buildOp('traffic.set', form({ amount: '50' }), 'UTC')?.params).toEqual({ gb: '50' });
    expect(buildOp('time.restart', form({ amount: '30' }), 'UTC')?.params).toEqual({ days: '30' });
    expect(buildOp('time.restart', form({ amount: '0' }), 'UTC')).toBeNull();
  });

  it('sets the expiry at the start of the day in the operator zone', () => {
    const date = { year: 2026, month: 3, day: 21 };
    expect(buildOp('time.set', form({ date }), 'Asia/Tehran')?.params).toEqual({
      date: '2026-03-20T20:30:00Z',
    });
    expect(buildOp('time.set', form({}), 'UTC')).toBeNull();
  });

  it('needs the text names and notes take', () => {
    expect(buildOp('name.prefix', form({ text: '  ' }), 'UTC')).toBeNull();
    expect(buildOp('name.prefix', form({ text: 'old_' }), 'UTC')?.params).toEqual({ text: 'old_' });
    expect(buildOp('name.replace', form({ from: 'shop', to: '' }), 'UTC')?.params).toEqual({
      from: 'shop',
      to: '',
    });
    expect(buildOp('name.replace', form({ to: 'x' }), 'UTC')).toBeNull();
  });

  it('needs targets, and protocols plus the warning read for new credentials', () => {
    expect(buildOp('place.targets', form({}), 'UTC')).toBeNull();
    expect(buildOp('place.targets', form({ ids: ['3', '5'] }), 'UTC')?.params).toEqual({
      ids: '3,5',
    });
    expect(buildOp('cred.new', form({ protocols: ['vless'] }), 'UTC')).toBeNull();
    expect(
      buildOp('cred.new', form({ protocols: ['vless', 'trojan'], understood: true }), 'UTC'),
    ).toEqual({ kind: 'cred.new', params: { protocols: 'vless,trojan' } });
    expect(buildOp('cred.new', form({ understood: true }), 'UTC')).toBeNull();
  });

  it('builds the operations without fields as they are', () => {
    expect(buildOp('skip', form({}), 'UTC')).toEqual({ kind: 'skip' });
    expect(buildOp('status.set', form({ status: 'disabled' }), 'UTC')?.params).toEqual({
      status: 'disabled',
    });
  });
});

describe('buildScope', () => {
  it('reads who a change reaches', () => {
    expect(buildScope(emptyScopeForm)).toEqual({ kind: 'all' });
    expect(buildScope({ ...emptyScopeForm, kind: 'category' })).toBeNull();
    expect(buildScope({ ...emptyScopeForm, kind: 'category', category: 'vip' })).toEqual({
      kind: 'category',
      id: 'vip',
    });
    expect(buildScope({ ...emptyScopeForm, kind: 'filter', value: '' })).toBeNull();
    expect(buildScope({ ...emptyScopeForm, kind: 'filter', value: 'vip' })).toEqual({
      kind: 'filter',
      condition: { field: 'username', op: 'contains', value: 'vip' },
    });
    expect(buildScope({ ...emptyScopeForm, kind: 'users' })).toBeNull();
    expect(buildScope({ ...emptyScopeForm, kind: 'users', keys: ['a'] })).toEqual({
      kind: 'users',
      keys: ['a'],
    });
  });
});
