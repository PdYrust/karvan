import { describe, expect, it } from 'vitest';
import { createTranslate } from '../../../i18n/core';
import { overviewMessages } from '../messages';
import { failureFacts } from './safe-mode-facts';

const en = createTranslate(overviewMessages.en, 'en');
const fa = createTranslate(overviewMessages.fa, 'fa');

describe('failureFacts', () => {
  it('names the undo when Safe Mode is on', () => {
    expect(failureFacts({ enabled: true, rollback: 'undo' }, en)).toEqual([
      { label: 'If a migration fails', value: 'Remove what it wrote' },
    ]);
  });

  it('names the backup when Safe Mode is on', () => {
    expect(failureFacts({ enabled: true, rollback: 'backup' }, en)).toEqual([
      { label: 'If a migration fails', value: 'Put the whole backup back' },
    ]);
  });

  it('says what it wrote stays when Safe Mode is off, whatever the saved choice', () => {
    expect(failureFacts({ enabled: false, rollback: 'undo' }, en)[0]).toEqual({
      label: 'If a migration fails',
      value: 'What it wrote stays',
    });
    expect(failureFacts({ enabled: false, rollback: 'backup' }, en)[0]).toEqual({
      label: 'If a migration fails',
      value: 'What it wrote stays',
    });
  });

  it('keeps the saved choice as what happens with Safe Mode on', () => {
    expect(failureFacts({ enabled: false, rollback: 'backup' }, en)[1]).toEqual({
      label: 'With Safe Mode on',
      value: 'Put the whole backup back',
    });
  });

  it('reads the same in Persian', () => {
    expect(failureFacts({ enabled: true, rollback: 'backup' }, fa)).toEqual([
      { label: 'اگر انتقال شکست بخورد', value: 'برگرداندن کامل پشتیبان' },
    ]);
    expect(failureFacts({ enabled: false, rollback: 'undo' }, fa)).toEqual([
      { label: 'اگر انتقال شکست بخورد', value: 'هر چه نوشته باشد میماند' },
      { label: 'اگر حالت امن روشن باشد', value: 'حذف چیزهایی که نوشته' },
    ]);
  });
});
