import { useEffect, useRef } from 'react';
import { drawWall, makePalette } from './wall-draw';
import { buildWall, type Wall } from './wall-geometry';

// The wall starts from this moment, in seconds; with reduced motion it stays there.
const start = 42;

/** A theme color as the canvas can use it. */
function token(el: HTMLElement, name: string): string {
  el.style.color = `var(${name})`;
  const color = getComputedStyle(el).color;
  el.style.color = '';
  return color;
}

const readPalette = (el: HTMLElement) =>
  makePalette(
    token(el, '--color-kumo-brand'),
    token(el, '--color-on-brand'),
    token(el, '--color-wall-shade'),
  );

/** The firewall on the sign-in page: hexagonal columns sliding slowly in and out. */
export default function LoginWall() {
  const canvas = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const el = canvas.current;
    const ctx = el?.getContext('2d');
    if (!el || !ctx) return;
    const still = matchMedia('(prefers-reduced-motion: reduce)').matches;
    let palette = readPalette(el);
    let wall: Wall | null = null;
    let [w, h] = [0, 0];
    let frameId = 0;
    let last = 0;
    let visible = false;
    const begin = performance.now();

    const render = (now: number) => {
      if (!wall) return;
      drawWall(ctx, wall, w, h, start + (still ? 0 : (now - begin) / 1000), palette);
      // The chunk arrives after the form: the wall fades in instead of popping up.
      el.style.opacity = '1';
    };
    const loop = (now: number) => {
      frameId = requestAnimationFrame(loop);
      // About 30 frames a second is smooth enough for slow motion and spares slow machines.
      if (now - last < 32) return;
      last = now;
      render(now);
    };
    const play = () => {
      if (!still && visible && !frameId) frameId = requestAnimationFrame(loop);
    };
    const pause = () => {
      cancelAnimationFrame(frameId);
      frameId = 0;
    };
    // The brand blue is a little darker in the dark theme. The attribute is
    // watched instead of using the theme hook, which would split the chunk
    // shared with the app and cost every page about 1 KB.
    const theme = new MutationObserver(() => {
      palette = readPalette(el);
      render(performance.now());
    });
    theme.observe(document.documentElement, { attributeFilter: ['data-mode'] });

    const resize = new ResizeObserver(([entry]) => {
      if (!entry) return;
      [w, h] = [entry.contentRect.width, entry.contentRect.height];
      wall = w && h ? buildWall(w, h) : null;
      // At 1.5 pixels per point the edges stay sharp and a frame paints about
      // half the pixels it would at 2.
      const ratio = Math.min(devicePixelRatio || 1, 1.5);
      el.width = Math.round(w * ratio);
      el.height = Math.round(h * ratio);
      ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
      render(performance.now());
    });
    // Off screen it does not draw at all.
    const seen = new IntersectionObserver(([entry]) => {
      visible = entry?.isIntersecting ?? false;
      if (visible) play();
      else pause();
    });
    resize.observe(el);
    seen.observe(el);
    return () => {
      pause();
      resize.disconnect();
      seen.disconnect();
      theme.disconnect();
    };
  }, []);

  return (
    <canvas
      ref={canvas}
      aria-hidden="true"
      // Mirrored in RTL, so the wall recedes toward the form in both languages.
      className="pointer-events-none absolute inset-0 size-full opacity-0 transition-opacity duration-700 ease-out rtl:-scale-x-100"
    />
  );
}
