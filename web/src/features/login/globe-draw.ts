import { arcPoint, type Land, type Vec } from './globe-math';

/** One frame of the globe: a square canvas `size` CSS pixels wide. */
export interface Frame {
  ctx: CanvasRenderingContext2D;
  size: number;
  color: string;
  lon0: number;
  tilt: number;
}

const radius = (size: number) => size / 2 - 4;

/** Where a view-space point lands on the canvas, in CSS pixels. */
export function toScreen(size: number, v: Vec): [number, number] {
  const r = radius(size);
  return [size / 2 + v[0] * r, size / 2 - v[1] * r];
}

export function drawLand({ ctx, size, color, lon0, tilt }: Frame, land: Land) {
  const r = radius(size);
  const c = size / 2;
  const scale = r / 420;
  ctx.clearRect(0, 0, size, size);
  ctx.fillStyle = color;
  ctx.strokeStyle = color;

  ctx.beginPath();
  ctx.arc(c, c, r, 0, Math.PI * 2);
  ctx.globalAlpha = 0.06;
  ctx.fill();
  ctx.globalAlpha = 0.2;
  ctx.lineWidth = 1.5;
  ctx.stroke();

  const far = new Path2D();
  const mid = new Path2D();
  const near = new Path2D();
  const cos0 = Math.cos(lon0);
  const sin0 = Math.sin(lon0);
  const st = Math.sin(tilt);
  const ct = Math.cos(tilt);
  for (const d of land) {
    // cos and sin of (lon - lon0) from the stored sines and cosines.
    const cd = d.cosLon * cos0 + d.sinLon * sin0;
    const z = st * d.sinLat + ct * d.cosLat * cd;
    if (z <= 0) continue;
    const sd = d.sinLon * cos0 - d.cosLon * sin0;
    const x = c + d.cosLat * sd * r;
    const y = c - (ct * d.sinLat - st * d.cosLat * cd) * r;
    const size = (0.7 + 1.5 * z) * scale;
    const p = z < 0.4 ? far : z < 0.75 ? mid : near;
    p.moveTo(x + size, y);
    p.arc(x, y, size, 0, Math.PI * 2);
  }
  // Dots nearer the viewer are brighter.
  ctx.globalAlpha = 0.26;
  ctx.fill(far);
  ctx.globalAlpha = 0.48;
  ctx.fill(mid);
  ctx.globalAlpha = 0.74;
  ctx.fill(near);
  ctx.globalAlpha = 1;
}

const lift = 0.22;

/**
 * The route from a to b: a dashed arc drawn up to `grow` (0 to 1), and packets
 * (users on their way) at the given positions along it.
 */
export function drawRoute(
  f: Frame,
  a: Vec,
  b: Vec,
  grow: number,
  alpha: number,
  packets: number[],
) {
  const { ctx, size, color } = f;
  if (alpha <= 0) return;
  const scale = radius(size) / 420;
  ctx.strokeStyle = color;
  ctx.fillStyle = color;

  if (grow > 0) {
    ctx.beginPath();
    const steps = 48;
    for (let s = 0; s <= steps * grow; s++) {
      const [x, y] = toScreen(size, arcPoint(a, b, s / steps, lift));
      if (s === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    }
    ctx.globalAlpha = 0.85 * alpha;
    ctx.lineWidth = 2.5 * scale;
    ctx.lineCap = 'round';
    ctx.setLineDash([1, 9 * scale]);
    ctx.stroke();
    ctx.setLineDash([]);
  }

  for (const t of packets) {
    // A bright head with a short fading tail behind it.
    for (let k = 0; k < 4; k++) {
      const tk = t - k * 0.035;
      if (tk <= 0) break;
      const [x, y] = toScreen(size, arcPoint(a, b, tk, lift));
      ctx.globalAlpha = alpha * (1 - k * 0.24);
      ctx.beginPath();
      ctx.arc(x, y, (5 - k) * scale, 0, Math.PI * 2);
      ctx.fill();
    }
  }
  ctx.globalAlpha = 1;
}
