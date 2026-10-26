import { describe, expect, it } from 'vitest';
import { qrPath } from './qr-path';

const uri = 'otpauth://totp/Karvan:operator?secret=JBSWY3DPEHPK3PXP&issuer=Karvan';

describe('qrPath', () => {
  it('draws a square code with its quiet zone', () => {
    const { size, d } = qrPath(uri);
    // Version 4 or 5 at level M for this length, plus four modules on each side.
    expect([33 + 8, 37 + 8]).toContain(size);
    expect(d.startsWith('M4 4h1v1h-1z')).toBe(true);
  });

  it('leaves the quiet zone empty', () => {
    const { size, d } = qrPath(uri);
    for (const m of d.matchAll(/M(\d+) (\d+)/g)) {
      const [x, y] = [Number(m[1]), Number(m[2])];
      expect(x >= 4 && y >= 4 && x < size - 4 && y < size - 4).toBe(true);
    }
  });

  it('gives the same text the same code', () => {
    expect(qrPath(uri)).toEqual(qrPath(uri));
    expect(qrPath(uri).d).not.toBe(qrPath(`${uri}x`).d);
  });
});
