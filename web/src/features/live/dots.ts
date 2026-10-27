/**
 * The moving dots of the live view, as plain math the scene draws each
 * frame (DESIGN.md, Live progress). Times are milliseconds on one clock,
 * such as performance.now().
 */

/** At most this many dots fly at once. */
export const maxDots = 24;
/** How long a dot takes from one panel to the other. */
export const travelMs = 1600;
/** The gap between dots of one chunk, so they leave as a short stream. */
export const staggerMs = 70;

/** forward: along the arc in the page direction; back: the other way; sink: off to the skipped users. */
export type Lane = 'forward' | 'back' | 'sink';

export interface Dot {
  lane: Lane;
  born: number;
}

/** A chunk too small for its share still sends a short stream, so it reads as a chunk. */
export const minChunkDots = 3;

/** How many dots a chunk sends: its share of the run, at least a few, within the cap. */
export function chunkDots(delta: number, total: number, inFlight: number): number {
  if (delta <= 0 || total <= 0) return 0;
  const want = Math.max(minChunkDots, Math.round((delta / total) * maxDots));
  return Math.max(0, Math.min(want, maxDots - inFlight));
}

/**
 * How many of n new dots go to the sink, so that over the run the sink gets
 * its share of every dot sent: skipped users of all users read.
 */
export function sinkDots(sent: number, sunk: number, n: number, skipShare: number): number {
  const due = Math.round((sent + n) * Math.min(1, Math.max(0, skipShare))) - sunk;
  return Math.min(n, Math.max(0, due));
}

/** n dots on one lane, leaving one after another from now. */
export function launch(now: number, n: number, lane: Lane): Dot[] {
  return Array.from({ length: n }, (_, i) => ({ lane, born: now + i * staggerMs }));
}

/** How far a dot has come, 0 to 1, eased in and out; below 0 before it leaves. */
export function progressOf(d: Dot, now: number): number {
  const t = (now - d.born) / travelMs;
  if (t < 0) return t;
  const c = Math.min(1, t);
  return c < 0.5 ? 4 * c * c * c : 1 - (-2 * c + 2) ** 3 / 2;
}

export interface Point {
  x: number;
  y: number;
}

export type Curve = readonly [Point, Point, Point, Point];

/** A point on a cubic Bezier curve. */
export function pointOn(c: Curve, t: number): Point {
  const u = 1 - t;
  const a = u * u * u;
  const b = 3 * u * u * t;
  const d = 3 * u * t * t;
  const e = t * t * t;
  return {
    x: a * c[0].x + b * c[1].x + d * c[2].x + e * c[3].x,
    y: a * c[0].y + b * c[1].y + d * c[2].y + e * c[3].y,
  };
}

/** The SVG path of a curve, rounded to a tenth of a pixel. */
export function pathOf(c: Curve): string {
  const p = (q: Point) =>
    `${String(Math.round(q.x * 10) / 10)} ${String(Math.round(q.y * 10) / 10)}`;
  return `M ${p(c[0])} C ${p(c[1])} ${p(c[2])} ${p(c[3])}`;
}

/** The sink's label sits at the bottom middle of the scene, this tall. */
export const sinkLabel = 28;

/**
 * The scene's two curves in a box of w by h pixels, start edge at x 0: the
 * arc from one panel to the other at mid height, where the tiles sit, and
 * the branch from the sending panel down to the sink's label.
 */
export function curvesFor(w: number, h: number, sink: boolean): { arc: Curve; sink: Curve | null } {
  const y = h / 2;
  const lift = Math.min(36, h * 0.24);
  const arc: Curve = [
    { x: 0, y },
    { x: w * 0.3, y: y - lift },
    { x: w * 0.7, y: y - lift },
    { x: w, y },
  ];
  if (!sink) return { arc, sink: null };
  const bottom = h - sinkLabel;
  return {
    arc,
    sink: [
      { x: 0, y },
      { x: w * 0.22, y },
      { x: w * 0.42, y: bottom },
      { x: w * 0.5, y: bottom },
    ],
  };
}

/** Where the sink's branch ends in a standing scene, as shares of its width and height. */
export const sinkEnd = { x: 0.68, y: 0.5 } as const;
/**
 * The curves of a standing scene (phones), w by h pixels: a straight line
 * down the middle from the sending panel above to the receiving one below,
 * and the sink's branch off it toward the end edge.
 */
export function curvesDown(
  w: number,
  h: number,
  sink: boolean,
): { arc: Curve; sink: Curve | null } {
  const x = w / 2;
  const arc: Curve = [
    { x, y: 0 },
    { x, y: h / 3 },
    { x, y: (2 * h) / 3 },
    { x, y: h },
  ];
  if (!sink) return { arc, sink: null };
  const end = { x: w * sinkEnd.x, y: h * sinkEnd.y };
  return {
    arc,
    sink: [{ x, y: 0 }, { x, y: h * 0.3 }, { x: end.x - w * 0.08, y: end.y }, end],
  };
}

/** Where a dot is now, or null when it has not left or has arrived. */
export function placeOf(d: Dot, now: number, arc: Curve, sink: Curve | null): Point | null {
  const t = progressOf(d, now);
  if (t < 0 || now - d.born >= travelMs) return null;
  if (d.lane === 'sink') return sink ? pointOn(sink, t) : null;
  return pointOn(arc, d.lane === 'back' ? 1 - t : t);
}
