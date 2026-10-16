import { landRows, landStep } from './globe-land';

export const deg = Math.PI / 180;

/** A land dot: latitude and longitude in radians, with their sines and cosines. */
export interface Dot {
  lat: number;
  lon: number;
  sinLat: number;
  cosLat: number;
  sinLon: number;
  cosLon: number;
}

export type Land = readonly Dot[];

export function buildLand(): Land {
  const land: Dot[] = [];
  landRows.forEach((row, r) => {
    const lat = (-80 + r * landStep) * deg;
    const [slots = 1, ...runs] = row;
    for (let k = 0; k + 1 < runs.length; k += 2) {
      const start = runs[k] ?? 0;
      const end = start + (runs[k + 1] ?? 0);
      for (let i = start; i < end; i++) {
        const lon = (-180 + (i * 360) / slots) * deg;
        land.push({
          lat,
          lon,
          sinLat: Math.sin(lat),
          cosLat: Math.cos(lat),
          sinLon: Math.sin(lon),
          cosLon: Math.cos(lon),
        });
      }
    }
  });
  return land;
}

export type Vec = [number, number, number];

/**
 * A point on the unit sphere in view space: x right, y up, z toward the viewer
 * (z > 0 is the visible side). The view looks at latitude `tilt`, longitude `lon0`.
 */
export function toView(lat: number, lon: number, lon0: number, tilt: number): Vec {
  const d = lon - lon0;
  const cl = Math.cos(lat);
  return [
    cl * Math.sin(d),
    Math.cos(tilt) * Math.sin(lat) - Math.sin(tilt) * cl * Math.cos(d),
    Math.sin(tilt) * Math.sin(lat) + Math.cos(tilt) * cl * Math.cos(d),
  ];
}

/** A point `t` of the way along the great circle from a to b, lifted off the surface. */
export function arcPoint(a: Vec, b: Vec, t: number, lift: number): Vec {
  const dot = Math.min(1, Math.max(-1, a[0] * b[0] + a[1] * b[1] + a[2] * b[2]));
  const w = Math.acos(dot);
  const s = Math.sin(w) || 1;
  const ka = Math.sin((1 - t) * w) / s;
  const kb = Math.sin(t * w) / s;
  const h = 1 + lift * Math.sin(Math.PI * t);
  return [(ka * a[0] + kb * b[0]) * h, (ka * a[1] + kb * b[1]) * h, (ka * a[2] + kb * b[2]) * h];
}

/** 0 before `from`, 1 after `to`, eased in between. */
export function ramp(t: number, from: number, to: number): number {
  const x = Math.min(1, Math.max(0, (t - from) / (to - from)));
  return x * x * (3 - 2 * x);
}
