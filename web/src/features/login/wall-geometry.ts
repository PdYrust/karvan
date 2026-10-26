/**
 * The firewall on the sign-in page: a wall of hexagonal columns, like the
 * channel caps of a reactor floor stood on end. Wall space: x along the wall,
 * y up, z out of the wall toward the viewer. One unit is a column's spacing
 * radius; the wall itself is the plane z = 0 and has no edges.
 */
export type Vec3 = readonly [number, number, number];

const deg = Math.PI / 180;
const root3 = Math.sqrt(3);
const dot = (a: Vec3, b: Vec3) => a[0] * b[0] + a[1] * b[1] + a[2] * b[2];
const unit = (a: Vec3): Vec3 => {
  const n = Math.hypot(a[0], a[1], a[2]);
  return [a[0] / n, a[1] / n, a[2] / n];
};

/** How far a column stands out of the wall, settled and fully out. */
export const reach = { min: 0.2, max: 2 };
/** A column's radius; the rest of the spacing is the gap around it. */
const radius = 0.9;
/** Light from the top start (top left before the RTL mirror), a little from the front. */
const light = unit([-0.5, 0.75, 0.45]);

/**
 * Brightness of each face under the light, 0 to 1: the six sides (side k runs
 * from corner k to corner k + 1, facing 60 + 60k degrees) and the cap.
 */
export const sideLight = Array.from({ length: 6 }, (_, k) => {
  const a = (60 + 60 * k) * deg;
  return 0.5 + 0.5 * dot([Math.cos(a), Math.sin(a), 0], light);
});
export const capLight = 0.5 + 0.5 * light[2];

export interface Camera {
  eye: Vec3;
  right: Vec3;
  up: Vec3;
  ahead: Vec3;
  focal: number;
  cx: number;
  cy: number;
}

/**
 * The view for a frame w by h CSS pixels: from the end side and a little from
 * below, looking across the wall, so its far corner is the top start.
 */
export function camera(w: number, h: number): Camera {
  const yaw = 30 * deg;
  const pitch = 12 * deg;
  const dist = 26;
  const eye: Vec3 = [
    dist * Math.sin(yaw) * Math.cos(pitch),
    -dist * Math.sin(pitch),
    dist * Math.cos(yaw) * Math.cos(pitch),
  ];
  const ahead = unit([-eye[0], -eye[1], -eye[2]]);
  const right = unit([-ahead[2], 0, ahead[0]]);
  const up: Vec3 = [
    right[1] * ahead[2] - right[2] * ahead[1],
    right[2] * ahead[0] - right[0] * ahead[2],
    right[0] * ahead[1] - right[1] * ahead[0],
  ];
  // A column in the middle of the frame is about a twelfth of its long side.
  const cell = Math.min(120, Math.max(40, Math.max(w, h) / 12));
  return { eye, right, up, ahead, focal: (cell * dist) / 2, cx: w / 2, cy: h / 2 };
}

/** A wall-space point in camera space: right, up and depth. */
export function toCamera(cam: Camera, p: Vec3): Vec3 {
  const v: Vec3 = [p[0] - cam.eye[0], p[1] - cam.eye[1], p[2] - cam.eye[2]];
  return [dot(v, cam.right), dot(v, cam.up), dot(v, cam.ahead)];
}

/** Where the ray through a screen point meets the wall. */
function onWall(cam: Camera, sx: number, sy: number): [number, number] {
  const a = (sx - cam.cx) / cam.focal;
  const b = (cam.cy - sy) / cam.focal;
  const ray = [0, 1, 2].map(
    (i) => (cam.ahead[i] ?? 0) + a * (cam.right[i] ?? 0) + b * (cam.up[i] ?? 0),
  );
  const t = -cam.eye[2] / (ray[2] ?? -1);
  return [cam.eye[0] + t * (ray[0] ?? 0), cam.eye[1] + t * (ray[1] ?? 0)];
}

export interface Column {
  /** Centre on the wall. */
  x: number;
  y: number;
  /** The six corners where the column leaves the wall, in camera space. */
  base: Float64Array;
  /** The same corners on the screen. */
  flat: Float64Array;
  /** The sides the eye can see. */
  sides: number[];
  /**
   * The column across each of those sides, by its index in the wall, when it
   * is drawn later (else -1): standing as far out, it hides the side.
   */
  across: number[];
  /** 0 for the nearest column in the frame, 1 for the furthest. */
  far: number;
  phase: number;
  speed: number;
}

export interface Wall {
  cam: Camera;
  /** Far to near, the order to draw them in. */
  columns: Column[];
  /** One unit out of the wall, in camera space. */
  out: Vec3;
}

/** A steady pseudo-random 0 to 1 per column, so the same column always moves the same way. */
function hash(i: number, j: number, salt: number): number {
  const s = Math.sin(i * 127.1 + j * 311.7 + salt * 74.7) * 43758.5453;
  return s - Math.floor(s);
}

const spot = (x: number, y: number) => `${Math.round(x * 10)},${Math.round(y * 10)}`;

/** The column at lattice cell i, j, or nothing when no part of it can reach the frame. */
function place(cam: Camera, out: Vec3, w: number, h: number, i: number, j: number) {
  const x = root3 * (i + (j & 1) / 2);
  const y = 1.5 * j;
  const base = new Float64Array(18);
  const flat = new Float64Array(12);
  let [minX, minY, maxX, maxY] = [Infinity, Infinity, -Infinity, -Infinity];
  for (let k = 0; k < 6; k++) {
    const a = (30 + 60 * k) * deg;
    const p = toCamera(cam, [x + radius * Math.cos(a), y + radius * Math.sin(a), 0]);
    base.set(p, 3 * k);
    for (const d of [0, reach.max]) {
      const z = p[2] + d * out[2];
      const sx = cam.cx + (cam.focal * (p[0] + d * out[0])) / z;
      const sy = cam.cy - (cam.focal * (p[1] + d * out[1])) / z;
      if (d === 0) flat.set([sx, sy], 2 * k);
      [minX, maxX] = [Math.min(minX, sx), Math.max(maxX, sx)];
      [minY, maxY] = [Math.min(minY, sy), Math.max(maxY, sy)];
    }
  }
  if (maxX < 0 || maxY < 0 || minX > w || minY > h) return null;
  const sides = [0, 1, 2, 3, 4, 5].filter((k) => {
    const a = (60 + 60 * k) * deg;
    const nx = Math.cos(a);
    const ny = Math.sin(a);
    const mid = (radius * root3) / 2;
    return nx * (cam.eye[0] - x - mid * nx) + ny * (cam.eye[1] - y - mid * ny) > 0;
  });
  return {
    x,
    y,
    base,
    flat,
    sides,
    across: [],
    far: 0,
    depth: toCamera(cam, [x, y, 0])[2],
    phase: 2 * Math.PI * hash(i, j, 1),
    speed: (2 * Math.PI) / (7 + 5 * hash(i, j, 2)),
    // Columns lean away from the point of the wall nearest the eye, so
    // drawing from the furthest of those first lets nearer ones cover them.
    key: (x - cam.eye[0]) ** 2 + (y - cam.eye[1]) ** 2,
  };
}

/** The columns a w by h frame shows, far to near. */
export function buildWall(w: number, h: number): Wall {
  const cam = camera(w, h);
  const out: Vec3 = [cam.right[2], cam.up[2], cam.ahead[2]];
  const corners = [onWall(cam, 0, 0), onWall(cam, w, 0), onWall(cam, 0, h), onWall(cam, w, h)];
  const xs = corners.map((c) => c[0]);
  const ys = corners.map((c) => c[1]);
  // Columns just outside the frame lean into it when they stand out.
  const pad = 3;
  const [j0, j1] = [
    Math.floor((Math.min(...ys) - pad) / 1.5),
    Math.ceil((Math.max(...ys) + pad) / 1.5),
  ];
  const [i0, i1] = [
    Math.floor((Math.min(...xs) - pad) / root3),
    Math.ceil((Math.max(...xs) + pad) / root3),
  ];
  const columns: (Column & { depth: number; key: number })[] = [];
  for (let j = j0; j <= j1; j++) {
    for (let i = i0; i <= i1; i++) {
      const c = place(cam, out, w, h, i, j);
      if (c) columns.push(c);
    }
  }
  columns.sort((a, b) => b.key - a.key);
  const depths = columns.map((c) => c.depth);
  const near = Math.min(...depths);
  const span = Math.max(...depths) - near || 1;
  for (const c of columns) c.far = (c.depth - near) / span;
  const at = new Map(columns.map((c, n) => [spot(c.x, c.y), n]));
  for (const [n, c] of columns.entries()) {
    c.across = c.sides.map((k) => {
      const a = (60 + 60 * k) * deg;
      const m = at.get(spot(c.x + root3 * Math.cos(a), c.y + root3 * Math.sin(a))) ?? -1;
      return m > n ? m : -1;
    });
  }
  return { cam, columns, out };
}

/**
 * How far a column is out at t seconds, 0 (settled) to 1 (fully out): two slow
 * waves across the wall and the column's own rhythm, eased so it rests at both ends.
 */
export function lift(c: Pick<Column, 'x' | 'y' | 'phase' | 'speed'>, t: number): number {
  const wave = Math.sin(0.42 * c.x + 0.26 * c.y - 0.45 * t);
  const swell = Math.sin(-0.21 * c.x + 0.37 * c.y - 0.31 * t + 1.3);
  const own = Math.sin(c.speed * t + c.phase);
  const v = 0.5 + 0.5 * (0.4 * wave + 0.25 * swell + 0.35 * own);
  return v * v * (3 - 2 * v);
}
