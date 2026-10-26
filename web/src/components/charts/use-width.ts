import { useEffect, useRef, useState, type RefObject } from 'react';

/** The width of an element in CSS pixels, kept up to date as the layout changes. 0 until measured. */
export function useWidth<T extends HTMLElement>(): [RefObject<T | null>, number] {
  const [ref, size] = useSize<T>();
  return [ref, size.width];
}

/** The width and height of an element in CSS pixels, kept up to date. 0 until measured. */
export function useSize<T extends HTMLElement>(): [
  RefObject<T | null>,
  { width: number; height: number },
] {
  const ref = useRef<T>(null);
  const [size, setSize] = useState({ width: 0, height: 0 });
  useEffect(() => {
    const el = ref.current;
    if (!el || typeof ResizeObserver === 'undefined') return;
    // The observer also reports the first size, right after observe().
    const ro = new ResizeObserver(([entry]) => {
      if (!entry) return;
      const width = Math.floor(entry.contentRect.width);
      const height = Math.floor(entry.contentRect.height);
      setSize((s) => (s.width === width && s.height === height ? s : { width, height }));
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  return [ref, size];
}
