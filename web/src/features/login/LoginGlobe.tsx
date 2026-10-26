import { useEffect, useRef } from 'react';
import { PanelLogo, panelNames, type PanelKind } from '../../components/panels/PanelLogo';
import { drawLand, drawRoute, toScreen, type Frame } from './globe-draw';
import { buildLand, deg, ramp, toView, type Land, type Vec } from './globe-math';
import { sceneAt, sceneLength, type Scene } from './globe-scenes';

const tilt = 22 * deg;
// The globe turns east, about one turn a minute.
const turn = (6 * deg) / 1000;
const startLon = 70 * deg;
const kinds: readonly PanelKind[] = ['marzban', 'pasarguard'];

interface Place {
  lat: number;
  lon: number;
}

/**
 * Two land dots for a scene's panels: one to the west of the middle and one
 * near it, so both drift across the front while the scene plays.
 */
function pickPlaces(land: Land, lon0: number, seed: number): [Place, Place] {
  const west: Place[] = [];
  const middle: Place[] = [];
  for (const dot of land) {
    const lat = dot.lat / deg;
    if (lat < 8 || lat > 56) continue;
    // Degrees east of the middle of the view, from -180 to 180.
    const d = (((((dot.lon - lon0) / deg + 540) % 360) + 360) % 360) - 180;
    if (d > -52 && d < -30) west.push(dot);
    else if (d > -10 && d < 8) middle.push(dot);
  }
  // Over open sea there may be no land in a window; then the panel floats there.
  const at = (list: Place[], k: number, d: number): Place =>
    list[Math.floor(k * list.length)] ?? { lat: 30 * deg, lon: lon0 + d * deg };
  const a = at(west, (seed * 0.618) % 1, -40);
  const b = at(middle, (seed * 0.382 + 0.5) % 1, 0);
  // Alternate which side sends, so users travel both ways across the screen.
  return seed % 2 ? [b, a] : [a, b];
}

/** Packet start times (ms into the scene): a move sends many, a merge a few. */
const packetStarts = (scene: Scene) =>
  Array.from({ length: scene.kind === 'move' ? 6 : 3 }, (_, i) => 1000 + i * 420);
const packetTime = 1400;

/** The spinning dotted globe on the sign-in page, acting out moves and merges. */
export default function LoginGlobe() {
  const wrap = useRef<HTMLDivElement>(null);
  const canvas = useRef<HTMLCanvasElement>(null);
  const stops = useRef<Partial<Record<PanelKind, HTMLDivElement | null>>>({});

  useEffect(() => {
    const box = wrap.current;
    const el = canvas.current;
    const ctx = el?.getContext('2d');
    if (!box || !el || !ctx) return;
    const land = buildLand();
    const still = matchMedia('(prefers-reduced-motion: reduce)').matches;
    const color = getComputedStyle(el).color;
    let size = 0;
    let shown = -1;
    let places: [Place, Place] = pickPlaces(land, startLon, 0);
    let frameId = 0;
    let last = 0;
    let visible = false;
    const begin = performance.now();

    const placeStop = (kind: PanelKind, v: Vec, opacity: number, scale: number) => {
      const node = stops.current[kind];
      if (!node) return;
      const [x, y] = toScreen(size, v);
      // Hide a panel as it turns away to the far side.
      const o = opacity * ramp(v[2], 0.02, 0.2);
      node.style.opacity = o.toFixed(3);
      node.style.transform = `translate(${x}px, ${y}px) translate(-50%, -50%) scale(${scale.toFixed(3)})`;
    };

    const render = (elapsed: number) => {
      if (!size) return;
      const lon0 = startLon - elapsed * turn;
      const index = Math.floor(elapsed / sceneLength);
      const scene = sceneAt(index);
      if (index !== shown) {
        shown = index;
        places = pickPlaces(land, lon0, index);
      }
      const t = elapsed - index * sceneLength;
      const frame: Frame = { ctx, size, color, lon0, tilt };
      drawLand(frame, land);

      const a = toView(places[0].lat, places[0].lon, lon0, tilt);
      const b = toView(places[1].lat, places[1].lon, lon0, tilt);
      const appear = ramp(t, 0, 600);
      const fade = 1 - ramp(t, sceneLength - 800, sceneLength - 150);
      const starts = packetStarts(scene);
      // In a move the old panel goes away once the last user has arrived.
      const done = (starts.at(-1) ?? 0) + packetTime;
      const retire = scene.kind === 'move' ? ramp(t, done, done + 700) : 0;
      const packets = starts.map((s) => (t - s) / packetTime).filter((p) => p > 0 && p < 1);
      // The receiving panel swells a little as each user arrives.
      const pulse = Math.max(0, ...starts.map((s) => 1 - Math.abs(t - s - packetTime) / 260));
      drawRoute(frame, a, b, ramp(t, 500, 1300), fade * (1 - 0.7 * retire), packets);

      const grow = 0.6 + 0.4 * appear;
      placeStop(scene.from, a, appear * fade * (1 - retire), grow * (1 - 0.45 * retire));
      placeStop(scene.to, b, appear * fade, grow * (1 + 0.12 * pulse));
    };

    const loop = (now: number) => {
      frameId = requestAnimationFrame(loop);
      // About 30 frames a second is smooth enough and spares slow machines.
      if (now - last < 32) return;
      last = now;
      render(now - begin);
    };
    const play = () => {
      if (still) render(2600);
      else if (visible && !frameId) frameId = requestAnimationFrame(loop);
    };
    const pause = () => {
      cancelAnimationFrame(frameId);
      frameId = 0;
    };

    const resize = new ResizeObserver(([entry]) => {
      if (!entry) return;
      size = entry.contentRect.width;
      const ratio = Math.min(devicePixelRatio || 1, 2);
      el.width = Math.round(size * ratio);
      el.height = Math.round(size * ratio);
      ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
      if (still || !frameId) render(still ? 2600 : performance.now() - begin);
    });
    // Off screen (the side is hidden on small screens) it does not draw at all.
    const seen = new IntersectionObserver(([entry]) => {
      visible = entry?.isIntersecting ?? false;
      if (visible) play();
      else pause();
    });
    resize.observe(box);
    seen.observe(box);
    return () => {
      pause();
      resize.disconnect();
      seen.disconnect();
    };
  }, []);

  return (
    <div
      ref={wrap}
      aria-hidden="true"
      className="pointer-events-none absolute start-[-5%] top-[44%] aspect-square w-[110%]"
    >
      <canvas ref={canvas} className="absolute inset-0 size-full text-on-brand" />
      {kinds.map((kind) => (
        <div
          key={kind}
          ref={(node) => {
            stops.current[kind] = node;
          }}
          className="absolute flex flex-col items-center gap-2 will-change-transform"
          // Canvas coordinates are physical; the globe is not mirrored in RTL.
          style={{ left: 0, top: 0, opacity: 0 }}
        >
          <span className="grid size-14 place-items-center rounded-xl bg-on-brand text-(--color-kumo-brand) ring-4 ring-on-brand/25">
            <PanelLogo kind={kind} size={36} onLight />
          </span>
          <span className="rounded-md bg-kumo-brand px-2 text-sm font-medium text-on-brand">
            {panelNames[kind]}
          </span>
        </div>
      ))}
    </div>
  );
}
