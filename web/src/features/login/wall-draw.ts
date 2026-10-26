import { capLight, lift, reach, sideLight, type Wall } from './wall-geometry';

/** Steps per ramp: one per percent, so a cap that brightens never visibly jumps. */
const steps = 100;

/**
 * The wall's colors: the theme's brand blue mixed toward the shade and toward
 * the light. Every face is painted opaque in one fill, which keeps a frame
 * cheap on slow phones.
 */
export interface Palette {
  light: string;
  toShade: string[];
  toLight: string[];
}

type Rgb = [number, number, number];

/** A theme color in sRGB bytes, as the canvas paints it. */
function bytes(swatch: CanvasRenderingContext2D, color: string): Rgb {
  swatch.clearRect(0, 0, 1, 1);
  swatch.fillStyle = color;
  swatch.fillRect(0, 0, 1, 1);
  const [r = 0, g = 0, b = 0] = swatch.getImageData(0, 0, 1, 1).data;
  return [r, g, b];
}

// The mixes of two theme colors, written out for the canvas.
const hex = (c: Rgb) => `#${c.map((v) => Math.round(v).toString(16).padStart(2, '0')).join('')}`;

const ramp = (a: Rgb, b: Rgb) =>
  Array.from({ length: steps + 1 }, (_, i) =>
    hex([0, 1, 2].map((k) => (a[k] ?? 0) + (((b[k] ?? 0) - (a[k] ?? 0)) * i) / steps) as Rgb),
  );

/** The palette from the three theme colors (any CSS color the canvas reads). */
export function makePalette(face: string, light: string, shade: string): Palette {
  const swatch = document.createElement('canvas').getContext('2d', { willReadFrequently: true });
  if (!swatch) return { light, toShade: [face], toLight: [face] };
  const f = bytes(swatch, face);
  return { light, toShade: ramp(f, bytes(swatch, shade)), toLight: ramp(f, bytes(swatch, light)) };
}

const pick = (list: string[], mix: number) =>
  list[Math.round(Math.min(1, mix) * (list.length - 1))] ?? '';

/** The brand blue toward the shade (0 to 1) or, below 0, toward the light. */
export const tone = (p: Palette, mix: number) =>
  mix < 0 ? pick(p.toLight, -mix) : pick(p.toShade, mix);

/**
 * How much shade a face shows when it has `shade` of its own and is `far`
 * (0 to 1) away: far columns sink into a deeper blue and lose most of their
 * contrast, so the text over the far corner stays calm and readable.
 */
export const depthTone = (shade: number, far: number) => {
  const dim = 0.4 * far;
  return dim + (1 - dim) * (1 - 0.6 * far) * shade;
};

// How deep each side sits in shade: none toward the light, most away from it.
const sideShade = sideLight.map((b) => 0.75 * Math.max(0, (capLight - b) / capLight));
// The gaps between the columns.
const gap = 0.65;

// Corners 0 to 5 are where a column leaves the wall, 6 to 11 its cap.
const cap = [6, 7, 8, 9, 10, 11];
const side = cap.map((_, k) => [k, (k + 1) % 6, 6 + ((k + 1) % 6), 6 + k]);
// The cap's edges toward the light (sides 1 and 2) catch it.
const litEdge = [7, 8, 9];

const at = (pts: Float64Array, i: number) => pts[i] ?? 0;

function trace(ctx: CanvasRenderingContext2D, pts: Float64Array, order: readonly number[]) {
  ctx.beginPath();
  for (let n = 0; n < order.length; n++) {
    const i = order[n] ?? 0;
    if (n === 0) ctx.moveTo(at(pts, 2 * i), at(pts, 2 * i + 1));
    else ctx.lineTo(at(pts, 2 * i), at(pts, 2 * i + 1));
  }
}

const cos30 = Math.cos(Math.PI / 6);

/**
 * The round plug in the middle of a cap, as on a reactor's channel caps: a
 * circle drawn through the cap's own axes, so it sits in the same perspective.
 */
function plug(ctx: CanvasRenderingContext2D, pts: Float64Array) {
  let [mx, my] = [0, 0];
  for (let k = 12; k < 24; k += 2) {
    mx += at(pts, k) / 6;
    my += at(pts, k + 1) / 6;
  }
  // One cap radius along the wall (corners 0 and 5 against 2 and 3) and up it (1 against 4).
  const ax = (at(pts, 12) + at(pts, 22) - at(pts, 16) - at(pts, 18)) / (4 * cos30);
  const ay = (at(pts, 13) + at(pts, 23) - at(pts, 17) - at(pts, 19)) / (4 * cos30);
  const bx = (at(pts, 14) - at(pts, 20)) / 2;
  const by = (at(pts, 15) - at(pts, 21)) / 2;
  ctx.save();
  ctx.transform(ax, ay, bx, by, mx, my);
  ctx.beginPath();
  ctx.arc(0, 0, 0.4, 0, 2 * Math.PI);
  // The path keeps its place, and the line is drawn in screen pixels.
  ctx.restore();
}

/** One frame of the wall at t seconds, on a canvas whose transform is in CSS pixels. */
export function drawWall(
  ctx: CanvasRenderingContext2D,
  wall: Wall,
  w: number,
  h: number,
  t: number,
  p: Palette,
) {
  const { cam, columns, out } = wall;
  // The wall between the columns, deep in shade. The far corner is the top start.
  const deep = ctx.createLinearGradient(0, 0, w, h);
  deep.addColorStop(0, tone(p, depthTone(gap, 1)));
  deep.addColorStop(1, tone(p, depthTone(gap, 0)));
  ctx.globalAlpha = 1;
  ctx.fillStyle = deep;
  ctx.fillRect(0, 0, w, h);
  ctx.lineWidth = 1;
  ctx.lineJoin = 'round';
  ctx.strokeStyle = p.light;

  const pts = new Float64Array(24);
  const ups = columns.map((c) => lift(c, t));
  for (let n = 0; n < columns.length; n++) {
    const c = columns[n];
    const up = ups[n] ?? 0;
    if (!c) continue;
    const d = reach.min + (reach.max - reach.min) * up;
    pts.set(c.flat);
    for (let k = 0; k < 6; k++) {
      const x = at(c.base, 3 * k) + d * out[0];
      const y = at(c.base, 3 * k + 1) + d * out[1];
      const z = at(c.base, 3 * k + 2) + d * out[2];
      pts[12 + 2 * k] = cam.cx + (cam.focal * x) / z;
      pts[13 + 2 * k] = cam.cy - (cam.focal * y) / z;
    }
    ctx.globalAlpha = 1;
    for (let s = 0; s < c.sides.length; s++) {
      const k = c.sides[s] ?? 0;
      // A neighbour that stands as far out, drawn later, hides this side.
      if ((ups[c.across[s] ?? -1] ?? -1) >= up) continue;
      trace(ctx, pts, side[k] ?? []);
      ctx.fillStyle = tone(p, depthTone(sideShade[k] ?? 0, c.far));
      ctx.fill();
    }
    trace(ctx, pts, cap);
    // A settled cap sits in its neighbours' shade; one that stands out catches the light.
    ctx.fillStyle = tone(p, depthTone(up < 0.5 ? 0.3 * (0.5 - up) : -0.16 * (up - 0.5), c.far));
    ctx.fill();
    const edge = 1 - 0.6 * c.far;
    trace(ctx, pts, litEdge);
    ctx.globalAlpha = 0.4 * edge;
    ctx.stroke();
    plug(ctx, pts);
    ctx.globalAlpha = 0.12 * edge;
    ctx.stroke();
  }
  ctx.globalAlpha = 1;
}
