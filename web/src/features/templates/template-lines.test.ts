import { describe, expect, it } from 'vitest';
import { templateLines } from './template-lines';

describe('templateLines', () => {
  it('lists categories with changes, problem changes, fixes and the reset, in that order', () => {
    const lines = templateLines({
      name: 'Resellers',
      tree: [
        { id: 'a1', name: 'Active', conditions: [], ops: null },
        {
          id: 'a2',
          parentId: 'a1',
          name: 'Heavy',
          conditions: [],
          ops: [{ kind: 'traffic.add', params: { gb: '10' } }],
        },
      ],
      problems: { C26: [], C02: [{ kind: 'skip' }] },
      resolutions: { C11: { option: 'accept' }, C02: { option: 'suffix' } },
      resetSchedule: 'restart',
    });
    expect(lines.map((l) => l.kind)).toEqual(['category', 'problem', 'fix', 'fix', 'reset']);
    expect(lines[0]).toMatchObject({ kind: 'category', node: { id: 'a2' } });
    expect(lines.slice(2)).toEqual([
      { kind: 'fix', type: 'C02', option: 'suffix' },
      { kind: 'fix', type: 'C11', option: 'accept' },
      { kind: 'reset', schedule: 'restart' },
    ]);
  });
  it('has no lines for an empty template, and ignores a reset value it does not know', () => {
    expect(templateLines({ name: 'x', tree: null, resetSchedule: 'later' })).toEqual([]);
  });
});
