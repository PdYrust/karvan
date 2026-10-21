import { describe, expect, it } from 'vitest';
import { createTranslate } from '../../i18n/core';
import { scanMessages, scanToast } from './scan-messages';

const en = createTranslate(scanMessages.en, 'en');
const fa = createTranslate(scanMessages.fa, 'fa');

describe('scanToast', () => {
  it('counts the panels a scan found', () => {
    expect(scanToast([{ id: 1 }, { id: 2 }], en)).toEqual({
      title: 'Found 2 panels on this server.',
      description: undefined,
    });
    expect(scanToast([{ id: 1 }], en).title).toBe('Found 1 panel on this server.');
  });

  it('adds a line for folders it could not use', () => {
    expect(scanToast([{ id: 3 }, { id: 0 }], en)).toEqual({
      title: 'Found 1 panel on this server.',
      description: "1 folder can't be used yet. The Panels page says why.",
    });
  });

  it('says so when it found only folders it could not use', () => {
    expect(scanToast([{ id: 0 }, { id: 0 }], en)).toEqual({
      title: 'No panel on this server that Karvan can use yet.',
      description: "2 folders can't be used yet. The Panels page says why.",
    });
  });

  it('says so when it found nothing', () => {
    expect(scanToast([], en)).toEqual({
      title: 'No Marzban or PasarGuard found on this server.',
      description: undefined,
    });
  });

  it('writes the counts in Persian digits in Persian', () => {
    expect(scanToast([{ id: 1 }, { id: 2 }, { id: 0 }], fa)).toEqual({
      title: '۲ پنل روی این سرور پیدا شد.',
      description: '۱ پوشه هنوز قابل استفاده نیست. دلیلش در صفحه پنل ها آمده است.',
    });
  });

  it('has every word in both languages', () => {
    expect(Object.keys(scanMessages.fa).toSorted()).toEqual(
      Object.keys(scanMessages.en).toSorted(),
    );
  });
});
