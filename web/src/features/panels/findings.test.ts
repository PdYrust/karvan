import { describe, expect, it } from 'vitest';
import { createTranslate, type Message } from '../../i18n/core';
import { codeSpans } from '../../lib/format/code-spans';
import { findingDetail, findingMessages, findingText, type FindingKey } from './findings';

// Every finding code discovery reports (internal/discovery/scan.go and copies.go).
const codes = [
  'env_unreadable',
  'db_url_missing',
  'db_relative_path',
  'db_unsupported',
  'db_unreachable',
  'db_not_writable',
  'panel_copy',
  'not_a_panel',
  'unknown_revision',
  'schema_mismatch',
  'unknown_children',
  'xray_config_unreadable',
  'cert_unreadable',
  'path_relative',
];

describe('findingText', () => {
  it('names the sentence and its fix', () => {
    expect(findingText('db_unreachable')).toEqual({
      what: 'db_unreachable',
      fix: 'db_unreachable_fix',
    });
  });

  it('has a sentence and a fix for every code, in both languages', () => {
    for (const lang of ['en', 'fa'] as const) {
      const t = createTranslate(findingMessages[lang], lang);
      for (const code of codes) {
        const text = findingText(code);
        expect(text, code).toBeDefined();
        if (!text) continue;
        expect(t(text.what).length, `${lang} ${code}`).toBeGreaterThan(10);
        expect(t(text.fix).length, `${lang} ${code} fix`).toBeGreaterThan(10);
      }
    }
  });

  it('knows nothing of other codes, nor of the fix keys themselves', () => {
    expect(findingText('disk_full')).toBeUndefined();
    expect(findingText('db_unreachable_fix')).toBeUndefined();
  });

  it('marks the same machine values in both languages, each closed', () => {
    const marked = (text: Message) =>
      typeof text === 'string'
        ? codeSpans(text)
            .filter((s) => s.code)
            .map((s) => s.text)
        : [];
    for (const key of Object.keys(findingMessages.en) as FindingKey[]) {
      const en = findingMessages.en[key];
      const fa = findingMessages.fa[key];
      expect(String(en).split('`').length % 2, `en ${key}`).toBe(1);
      expect(String(fa).split('`').length % 2, `fa ${key}`).toBe(1);
      expect(marked(fa).toSorted(), key).toEqual(marked(en).toSorted());
    }
  });

  it('has no words for a code discovery does not report', () => {
    const keys = Object.keys(findingMessages.en).filter((k) => !k.endsWith('_fix'));
    expect(keys.toSorted()).toEqual(codes.toSorted());
    expect(Object.keys(findingMessages.fa).toSorted()).toEqual(
      Object.keys(findingMessages.en).toSorted(),
    );
  });
});

describe('findingDetail', () => {
  it('shows a detail that is a name, a revision, a table or a path', () => {
    expect(findingDetail({ code: 'path_relative', detail: './xray_config.json' })).toBe(
      './xray_config.json',
    );
    expect(findingDetail({ code: 'unknown_revision', detail: 'a1b2c3d4e5f6' })).toBe(
      'a1b2c3d4e5f6',
    );
    expect(findingDetail({ code: 'panel_copy', detail: '/opt/pasarguard' })).toBe(
      '/opt/pasarguard',
    );
    expect(findingDetail({ code: 'unknown_children', detail: 'logs.user_id -> users' })).toBe(
      'logs.user_id -> users',
    );
  });

  it('never shows raw error text', () => {
    expect(
      findingDetail({ code: 'db_unreachable', detail: 'dial tcp 127.0.0.1:3306: refused' }),
    ).toBeUndefined();
    expect(findingDetail({ code: 'env_unreadable', detail: 'permission denied' })).toBeUndefined();
    expect(findingDetail({ code: 'path_relative' })).toBeUndefined();
  });
});
