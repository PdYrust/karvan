import { useEffect, useRef, useState } from 'react';
import { useReducedMotion } from './reduced-motion';

/** One turn of Kumo's refresh icon (its `animate-refresh`, 0.5 s). */
export const refreshTurn = 500;
/** A switch's thumb slide. */
export const switchSlide = 150;

/**
 * How much longer motion that started `elapsed` ms ago must run: at least
 * `min` in all, and when `step` is given, up to the end of the current step
 * (a whole turn), never less than one.
 */
export function motionLeft(elapsed: number, min: number, step?: number): number {
  const steps = step ? Math.max(1, Math.ceil(elapsed / step)) * step : 0;
  return Math.max(0, Math.max(min, steps) - elapsed);
}

/**
 * `active`, held true until the motion it drives has run `min` ms (in whole
 * `step`s when given), so an instant answer does not cut it short. With
 * reduced motion nothing is held.
 */
export function useMinDuration(active: boolean, min: number, step?: number): boolean {
  const calm = useReducedMotion();
  const [held, setHeld] = useState(active);
  if (active && !held) setHeld(true);
  const since = useRef<number | null>(null);

  useEffect(() => {
    if (!held) {
      since.current = null;
      return;
    }
    since.current ??= Date.now();
    if (active) return;
    const id = setTimeout(() => setHeld(false), motionLeft(Date.now() - since.current, min, step));
    return () => clearTimeout(id);
  }, [held, active, min, step]);

  return calm ? active : active || held;
}
