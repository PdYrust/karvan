import { describe, expect, it } from 'vitest';
import { codeSpans } from './code-spans';

describe('codeSpans', () => {
  it('keeps a sentence without marks as one plain span', () => {
    expect(codeSpans("Karvan can't reach this database.")).toEqual([
      { text: "Karvan can't reach this database.", code: false },
    ]);
  });

  it('sets apart each value between backticks', () => {
    expect(codeSpans('Add `SQLALCHEMY_DATABASE_URL` to `.env`.')).toEqual([
      { text: 'Add ', code: false },
      { text: 'SQLALCHEMY_DATABASE_URL', code: true },
      { text: ' to ', code: false },
      { text: '.env', code: true },
      { text: '.', code: false },
    ]);
  });

  it('works the same in a Persian sentence', () => {
    expect(codeSpans('فایل `.env` این پنل')).toEqual([
      { text: 'فایل ', code: false },
      { text: '.env', code: true },
      { text: ' این پنل', code: false },
    ]);
  });

  it('drops empty pieces, such as a value at the very start', () => {
    expect(codeSpans('`/opt` is scanned')).toEqual([
      { text: '/opt', code: true },
      { text: ' is scanned', code: false },
    ]);
    expect(codeSpans('')).toEqual([]);
  });
});
