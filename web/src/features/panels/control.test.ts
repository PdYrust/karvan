import { describe, expect, it } from 'vitest';
import { createTranslate, type Message } from '../../i18n/core';
import { codeSpans } from '../../lib/format/code-spans';
import { controlMessages, controlText, type ControlKey } from './control';

// Every reason the server gives (controlReason in internal/httpapi/handlers/migration.go).
const reasons = ['turned_off', 'no_compose_file', 'no_panel_service', 'docker_denied'];

describe('controlText', () => {
  it('says so when Karvan can stop and start the panel', () => {
    expect(controlText({ available: true })).toEqual({
      label: 'available',
      hint: 'availableHint',
    });
  });

  it('gives each reason a short label, a sentence and a fix', () => {
    for (const reason of reasons) {
      expect(controlText({ available: false, reason })).toEqual({
        label: reason,
        hint: `${reason}_hint`,
        fix: `${reason}_fix`,
      });
    }
  });

  it('falls back to a general sentence for a reason it does not know', () => {
    expect(controlText({ available: false, reason: 'gone_fishing' })).toEqual({
      label: 'unavailable',
      hint: 'unavailableHint',
    });
    expect(controlText({ available: false })).toEqual({
      label: 'unavailable',
      hint: 'unavailableHint',
    });
  });

  it('marks the same machine values in both languages, each closed', () => {
    const marked = (text: Message) =>
      typeof text === 'string'
        ? codeSpans(text)
            .filter((s) => s.code)
            .map((s) => s.text)
        : [];
    for (const key of Object.keys(controlMessages.en) as ControlKey[]) {
      const en = controlMessages.en[key];
      const fa = controlMessages.fa[key];
      expect(String(en).split('`').length % 2, `en ${key}`).toBe(1);
      expect(String(fa).split('`').length % 2, `fa ${key}`).toBe(1);
      expect(marked(fa).toSorted(), key).toEqual(marked(en).toSorted());
    }
  });

  it('keeps the short labels plain, since tables show them as they are', () => {
    for (const key of ['available', 'unavailable', ...reasons] as ControlKey[]) {
      expect(controlMessages.en[key], key).not.toContain('`');
      expect(controlMessages.fa[key], key).not.toContain('`');
    }
  });

  it('has every word in both languages', () => {
    expect(Object.keys(controlMessages.fa).toSorted()).toEqual(
      Object.keys(controlMessages.en).toSorted(),
    );
    for (const lang of ['en', 'fa'] as const) {
      const t = createTranslate(controlMessages[lang], lang);
      for (const reason of reasons) {
        const text = controlText({ available: false, reason });
        expect(t(text.label).length, `${lang} ${reason}`).toBeGreaterThan(3);
        expect(t(text.hint).length, `${lang} ${reason} hint`).toBeGreaterThan(10);
        expect(text.fix && t(text.fix).length, `${lang} ${reason} fix`).toBeGreaterThan(10);
      }
    }
  });
});
