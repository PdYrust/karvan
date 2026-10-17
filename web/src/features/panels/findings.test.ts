import { describe, expect, it } from 'vitest';
import { findingMessages, findingText } from './findings';

describe('findingText', () => {
  it('names the sentence and its fix', () => {
    expect(findingText('db_unreachable')).toEqual({
      what: 'db_unreachable',
      fix: 'db_unreachable_fix',
    });
  });

  it('knows nothing of codes outside the list, nor of the fix keys themselves', () => {
    expect(findingText('schema_mismatch')).toBeUndefined();
    expect(findingText('db_unreachable_fix')).toBeUndefined();
  });

  it('has a fix for every finding, in both languages', () => {
    for (const lang of ['en', 'fa'] as const) {
      const keys = Object.keys(findingMessages[lang]);
      for (const k of keys.filter((x) => !x.endsWith('_fix'))) {
        expect(keys).toContain(`${k}_fix`);
      }
    }
  });
});
