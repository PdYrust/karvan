import { describe, expect, it } from 'vitest';
import { codeDigits, groupSecret } from './code';

describe('codeDigits', () => {
  it('keeps six digits and drops the rest', () => {
    expect(codeDigits('12 34-56')).toBe('123456');
    expect(codeDigits('1234567')).toBe('123456');
    expect(codeDigits('abc')).toBe('');
  });

  it('reads Persian and Arabic digits as the same digits', () => {
    expect(codeDigits('۱۲۳۴۵۶')).toBe('123456');
    expect(codeDigits('٠٩٨٧٦٥')).toBe('098765');
  });
});

describe('groupSecret', () => {
  it('splits a secret into groups of four', () => {
    expect(groupSecret('JBSWY3DPEHPK3PXP')).toBe('JBSW Y3DP EHPK 3PXP');
    expect(groupSecret('ABCDEF')).toBe('ABCD EF');
    expect(groupSecret('')).toBe('');
  });
});
