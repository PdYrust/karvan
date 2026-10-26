import { useEffect, useRef, useState } from 'react';
import { useReducedMotion } from './reduced-motion';

/** How long a skeleton stays once shown, so fast data still reads as arriving. */
export const minSkeleton = 400;

export interface FirstLoad<T> {
  /** The data, or undefined while the skeleton should still show. */
  data: T | undefined;
  /** The data is there but held back: show the skeleton, not an error. */
  holding: boolean;
  /** True once the data came after a skeleton: the content fades in. */
  reveal: boolean;
}

/**
 * Holds back the first data of a page or card until its skeleton has been on
 * screen for `min` ms. Data already there when the component mounts (from the
 * cache), refetches and polls pass straight through and never bring the
 * skeleton back. With reduced motion nothing is held and nothing fades.
 */
export function useFirstLoad<T>(value: T | undefined, min = minSkeleton): FirstLoad<T> {
  const calm = useReducedMotion();
  const loading = value === undefined;
  const [waiting, setWaiting] = useState(loading);
  const [revealed, setRevealed] = useState(false);
  // The data went away (a new key with nothing cached): its skeleton is held again.
  if (loading && !waiting) setWaiting(true);
  const since = useRef<number | null>(null);

  useEffect(() => {
    if (!waiting) {
      since.current = null;
      return;
    }
    since.current ??= Date.now();
    if (loading) return;
    const left = Math.max(0, min - (Date.now() - since.current));
    const id = setTimeout(() => {
      setWaiting(false);
      setRevealed(true);
    }, left);
    return () => clearTimeout(id);
  }, [waiting, loading, min]);

  const held = waiting && !calm;
  return {
    data: held ? undefined : value,
    holding: held && !loading,
    reveal: revealed && !calm,
  };
}
