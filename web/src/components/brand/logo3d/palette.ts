import type { Palette } from './scene';

/**
 * Reads the brand tokens and converts them to sRGB hex for WebGL. The tokens
 * are OKLCH colors, which three.js cannot parse, so a 1px canvas does the
 * conversion the same way the browser paints CSS.
 */
export function readPalette(): Palette {
  const style = getComputedStyle(document.documentElement);
  const ctx = document.createElement('canvas').getContext('2d', { willReadFrequently: true });
  const toHex = (token: string) => {
    const css = style.getPropertyValue(token).trim();
    if (!ctx || !css) return css;
    ctx.clearRect(0, 0, 1, 1);
    ctx.fillStyle = css;
    ctx.fillRect(0, 0, 1, 1);
    const [r = 0, g = 0, b = 0] = ctx.getImageData(0, 0, 1, 1).data;
    return '#' + [r, g, b].map((v) => v.toString(16).padStart(2, '0')).join('');
  };
  return {
    firouzeh: toHex('--karvan-firouzeh'),
    saffron: toHex('--karvan-saffron'),
    roadBed: toHex('--karvan-road'),
  };
}
