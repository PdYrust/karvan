import type { PanelKind } from '../api/panels';

/** The rules an admin password can break, named as the API names them (B10). */
export type PasswordRule =
  'password_short' | 'password_long' | 'password_weak' | 'password_has_name' | 'password_has_quote';

const minLength = 12;
// bcrypt reads 72 bytes.
const maxBytes = 72;
const byteLength = (s: string) => new TextEncoder().encode(s).length;

/**
 * The first rule a password breaks for the admin named username, or null.
 * It mirrors the policy Karvan checks before a run (PasarGuard's own,
 * panel/pasarguard/password.go); on Marzban only the length counts. The
 * server still judges the password when the run starts (D11).
 */
export function passwordRule(
  password: string,
  username: string,
  kind: PanelKind,
): PasswordRule | null {
  if (kind === 'marzban') {
    if (byteLength(password) < minLength) return 'password_short';
    return byteLength(password) > maxBytes ? 'password_long' : null;
  }
  const chars = [...password];
  let digits = 0;
  let upper = 0;
  let lower = 0;
  let other = 0;
  for (const c of chars) {
    if (/\p{Nd}/u.test(c)) digits++;
    else if (/\p{Lu}/u.test(c)) upper++;
    else if (/\p{Ll}/u.test(c)) lower++;
    else other++;
  }
  if (chars.length < minLength) return 'password_short';
  if (byteLength(password) > maxBytes) return 'password_long';
  if (digits < 2 || upper < 2 || lower < 2 || other < 1) return 'password_weak';
  if (username && password.toLowerCase().includes(username.toLowerCase())) {
    return 'password_has_name';
  }
  return password.includes('"') ? 'password_has_quote' : null;
}

// The sets Karvan's own generator uses, without look-alike characters.
const lowers = 'abcdefghijkmnpqrstuvwxyz';
const uppers = 'ABCDEFGHJKLMNPQRSTUVWXYZ';
const digitSet = '23456789';
const others = '!#$%&*+-=?@_';
const all = lowers + uppers + digitSet + others;
const generatedLength = 20;

/** Fills an array with random numbers: crypto.getRandomValues, or a seeded one in tests. */
export type RandomFill = (a: Uint32Array<ArrayBuffer>) => Uint32Array<ArrayBuffer>;
const cryptoFill: RandomFill = (a) => crypto.getRandomValues(a);

/** A random whole number below n, without the bias a plain remainder has. */
function below(n: number, fill: RandomFill): number {
  const limit = Math.floor(2 ** 32 / n) * n;
  const one = new Uint32Array(1);
  for (;;) {
    const v = fill(one)[0] ?? 0;
    if (v < limit) return v % n;
  }
}

/**
 * A 20 character password that meets the policy for the admin named
 * username, made in the browser (D11). It never starts with a character a
 * spreadsheet reads as a formula, as Karvan's own generator does.
 */
export function generatePassword(username: string, fill: RandomFill = cryptoFill): string {
  for (;;) {
    const chars = [lowers, lowers, uppers, uppers, digitSet, digitSet, others].map((set) =>
      set.charAt(below(set.length, fill)),
    );
    while (chars.length < generatedLength) chars.push(all.charAt(below(all.length, fill)));
    for (let i = chars.length - 1; i > 0; i--) {
      const j = below(i + 1, fill);
      [chars[i], chars[j]] = [chars[j] ?? '', chars[i] ?? ''];
    }
    const p = chars.join('');
    if (!'=+-@'.includes(p.charAt(0)) && passwordRule(p, username, 'pasarguard') === null) return p;
  }
}
