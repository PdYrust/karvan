import { encode } from 'uqr';

/** A QR code as one SVG path of unit squares, with the quiet zone of four modules around it. */
export function qrPath(text: string): { size: number; d: string } {
  const { data } = encode(text, { ecc: 'M', border: 4 });
  let d = '';
  data.forEach((row, y) => {
    row.forEach((dark, x) => {
      if (dark) d += `M${x} ${y}h1v1h-1z`;
    });
  });
  return { size: data.length, d };
}
