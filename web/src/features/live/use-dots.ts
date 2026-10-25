import { useEffect, useRef, useState, type RefObject } from 'react';
import { launch, maxDots, placeOf, travelMs, type Curve, type Dot, type Lane } from './dots';

/** About 30 frames a second is smooth enough and spares slow phones. */
const frameMs = 32;
/** A restore has no batches: while it runs, one dot leaves this often. */
const steadyMs = 360;

export interface Curves {
  arc: Curve;
  sink: Curve | null;
}

export type Lanes = Readonly<Record<Lane, string>>;

/**
 * The scene's one animation loop (DESIGN.md, Live progress), outside React.
 * Dots live in fixed slots of pre-drawn circles, moved by setting their
 * attributes, so a frame never renders a component. The loop runs only
 * while dots fly, the scene is on screen and the tab is visible; a steady
 * lane keeps one dot leaving every so often.
 */
export function createDots() {
  const slots: (Dot | null)[] = Array.from({ length: maxDots }, () => null);
  const circles: (SVGCircleElement | null)[] = [];
  const drawn: (Lane | null)[] = [];
  let curves: Curves | null = null;
  let lanes: Lanes | null = null;
  let steady: Lane | undefined;
  let frame = 0;
  let last = 0;
  let lastSteady = 0;
  let seen = true;

  function put(n: number, lane: Lane, now: number) {
    const free = slots.flatMap((d, i) => (d ? [] : [i]));
    launch(now, Math.min(n, free.length), lane).forEach((d, k) => {
      slots[free[k] ?? 0] = d;
    });
  }

  function place(i: number, now: number): boolean {
    const d = slots[i] ?? null;
    if (d && now - d.born >= travelMs) slots[i] = null;
    const c = circles[i];
    const p = d && curves ? placeOf(d, now, curves.arc, curves.sink) : null;
    if (c && !p) c.setAttribute('opacity', '0');
    if (c && d && p) {
      if (drawn[i] !== d.lane && lanes) {
        c.setAttribute('class', lanes[d.lane]);
        drawn[i] = d.lane;
      }
      c.setAttribute('cx', p.x.toFixed(1));
      c.setAttribute('cy', p.y.toFixed(1));
      c.setAttribute('opacity', '1');
    }
    return slots[i] !== null;
  }

  function loop(now: number) {
    frame = 0;
    if (!seen || document.hidden) return;
    if (steady && now - lastSteady >= steadyMs) {
      lastSteady = now;
      put(1, steady, now);
    }
    let busy = true;
    if (now - last >= frameMs) {
      last = now;
      busy = false;
      for (let i = 0; i < maxDots; i++) busy = place(i, now) || busy;
    }
    if (busy || steady) frame = requestAnimationFrame(loop);
  }

  function wake() {
    if (!frame && seen && !document.hidden) frame = requestAnimationFrame(loop);
  }

  return {
    /** The latest drawing and lanes; a steady lane starts the loop. */
    configure(next: Curves | null, nextLanes: Lanes, nextSteady?: Lane) {
      curves = next;
      lanes = nextLanes;
      steady = nextSteady;
      if (steady) wake();
    },
    /** Sends n dots on a lane, as many as fit. */
    send(n: number, lane: Lane) {
      if (n <= 0) return;
      put(n, lane, performance.now());
      wake();
    },
    inFlight: () => slots.filter(Boolean).length,
    /** One ref callback per slot. */
    circle: Array.from({ length: maxDots }, (_, i) => (node: SVGCircleElement | null) => {
      circles[i] = node;
    }),
    /** Pauses while the element is off screen or the tab is hidden; returns the cleanup. */
    observe(el: Element | null): () => void {
      // Dots due while the tab was hidden have arrived; the loop drops them.
      const onVisible = () => {
        if (!document.hidden) wake();
      };
      document.addEventListener('visibilitychange', onVisible);
      const io =
        el && typeof IntersectionObserver !== 'undefined'
          ? new IntersectionObserver(([entry]) => {
              seen = entry?.isIntersecting ?? true;
              if (seen) wake();
            })
          : null;
      if (el) io?.observe(el);
      return () => {
        document.removeEventListener('visibilitychange', onVisible);
        io?.disconnect();
        cancelAnimationFrame(frame);
        frame = 0;
      };
    },
  };
}

export type Dots = ReturnType<typeof createDots>;

/** The dots of one scene, and the element that tells whether it is on screen. */
export function useDots(
  curves: Curves | null,
  lanes: Lanes,
  steady?: Lane,
): [RefObject<HTMLDivElement | null>, Dots] {
  const box = useRef<HTMLDivElement>(null);
  const [dots] = useState(createDots);
  useEffect(() => dots.configure(curves, lanes, steady), [dots, curves, lanes, steady]);
  useEffect(() => dots.observe(box.current), [dots]);
  return [box, dots];
}
