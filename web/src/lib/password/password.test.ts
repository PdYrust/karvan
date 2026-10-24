import { describe, expect, it } from 'vitest';
import { generatePassword, passwordRule, type RandomFill } from './password';

describe('passwordRule on PasarGuard', () => {
  const rule = (p: string, name = 'reseller') => passwordRule(p, name, 'pasarguard');

  it('takes a password that meets every rule', () => {
    expect(rule('Ab12cD!efghij')).toBeNull();
  });

  it('counts characters, not bytes, for the minimum', () => {
    expect(rule('Ab12cD!efgh')).toBe('password_short');
    // Twelve characters, some of them two bytes long.
    expect(rule('ÄÖ12äö!efghi')).toBeNull();
  });

  it('counts bytes for the maximum, as bcrypt reads 72', () => {
    expect(rule(`AB12!${'x'.repeat(67)}`)).toBeNull();
    expect(rule(`AB12!${'x'.repeat(68)}`)).toBe('password_long');
    expect(rule(`AB12!${'ä'.repeat(34)}`)).toBe('password_long');
  });

  it('asks for 2 digits, 2 capitals, 2 small letters and 1 other character', () => {
    expect(rule('Ab1cD!efghijk')).toBe('password_weak');
    expect(rule('Ab12cd!efghij')).toBe('password_weak');
    expect(rule('AB12CD!EFGHIj')).toBe('password_weak');
    expect(rule('Ab12cDxefghij')).toBe('password_weak');
  });

  it('reads Persian digits as digits and Persian letters as other characters', () => {
    expect(rule('Ab۱۲cDسلامefgh')).toBeNull();
  });

  it('refuses the admin name in any case', () => {
    expect(rule('Ab12!xRESELLERx', 'reseller')).toBe('password_has_name');
    expect(rule('Ab12!xRESELLERx', '')).toBeNull();
  });

  it('refuses a double quote', () => {
    expect(rule('Ab12cD!efgh"j')).toBe('password_has_quote');
  });

  it('names the first broken rule in the order the panel checks them', () => {
    expect(rule('ab"')).toBe('password_short');
    expect(rule('reseller12"xyz')).toBe('password_weak');
  });
});

describe('passwordRule on Marzban', () => {
  it('only asks for 12 to 72 bytes', () => {
    expect(passwordRule('abcdefghijkl', 'reseller', 'marzban')).toBeNull();
    expect(passwordRule('abcdefghijk', 'reseller', 'marzban')).toBe('password_short');
    expect(passwordRule('x'.repeat(73), 'reseller', 'marzban')).toBe('password_long');
    expect(passwordRule('reseller"1234', 'reseller', 'marzban')).toBeNull();
  });
});

/** A seeded stand-in for crypto.getRandomValues, so a test can repeat a draw. */
function seeded(seed: number): RandomFill {
  let s = seed;
  return (a) => {
    for (let i = 0; i < a.length; i++) {
      s = (Math.imul(s, 1664525) + 1013904223) >>> 0;
      a[i] = s;
    }
    return a;
  };
}

describe('generatePassword', () => {
  it('meets the policy for the admin, 20 characters from the safe sets', () => {
    for (let i = 0; i < 300; i++) {
      const p = generatePassword('ab');
      expect(p).toHaveLength(20);
      expect(passwordRule(p, 'ab', 'pasarguard')).toBeNull();
      expect(p).toMatch(
        /^[abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!#$%&*+\-=?@_]+$/,
      );
    }
  });

  it('never starts with a character a spreadsheet reads as a formula', () => {
    for (let i = 0; i < 300; i++) expect('=+-@').not.toContain(generatePassword('shop').charAt(0));
  });

  it('draws from the random source it is given', () => {
    expect(generatePassword('shop', seeded(7))).toBe(generatePassword('shop', seeded(7)));
    expect(generatePassword('shop', seeded(7))).not.toBe(generatePassword('shop', seeded(8)));
  });
});
