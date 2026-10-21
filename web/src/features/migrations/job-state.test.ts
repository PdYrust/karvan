import { describe, expect, it } from 'vitest';
import { jobStates } from './job-state';
import { migrationsMessages } from './messages';

describe('job states', () => {
  it('gives every state a word in both languages', () => {
    for (const { word } of Object.values(jobStates)) {
      expect(migrationsMessages.en[word]).toBeTruthy();
      expect(migrationsMessages.fa[word]).toBeTruthy();
    }
  });

  it('reads both draft states as one', () => {
    expect(jobStates.created).toEqual(jobStates.checked);
  });

  it('marks a failed run as a problem and a finished one as done', () => {
    expect(jobStates.failed.tone).toBe('danger');
    expect(jobStates.done.tone).toBe('success');
    expect(jobStates.interrupted.tone).toBe('warning');
  });
});
