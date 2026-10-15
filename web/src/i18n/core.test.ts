import { describe, expect, it } from 'vitest';
import { createTranslate, directionOf, intlLocale } from './core';

const en = {
  hello: 'Hello {name}',
  users: { one: '{count} user', other: '{count} users' },
  plain: 'No params',
};

describe('createTranslate', () => {
  it('fills placeholders', () => {
    expect(createTranslate(en, 'en')('hello', { name: 'Sara' })).toBe('Hello Sara');
  });

  it('keeps unknown placeholders visible', () => {
    expect(createTranslate(en, 'en')('hello', {})).toBe('Hello {name}');
  });

  it('chooses plural forms', () => {
    const t = createTranslate(en, 'en');
    expect(t('users', { count: 1 })).toBe('1 user');
    expect(t('users', { count: 2340 })).toBe('2,340 users');
  });

  it('writes Persian digits for Persian', () => {
    const fa = { users: { one: '{count} کاربر', other: '{count} کاربر' } };
    expect(createTranslate(fa, 'fa')('users', { count: 340 })).toBe('۳۴۰ کاربر');
  });

  it('returns plain messages untouched', () => {
    expect(createTranslate(en, 'en')('plain')).toBe('No params');
  });
});

describe('locale helpers', () => {
  it('maps direction and Intl tags', () => {
    expect(directionOf('fa')).toBe('rtl');
    expect(directionOf('en')).toBe('ltr');
    expect(intlLocale('fa')).toBe('fa-IR');
    expect(intlLocale('en')).toBe('en-US');
  });
});
