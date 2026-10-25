import { useEffect, useRef, useState } from 'react';

/**
 * True once the element comes within `margin` of the viewport, and true from
 * then on. On a phone the overview's analytics sit far below the first screen,
 * so they load and draw only when the reader scrolls toward them.
 */
export function useNear<T extends Element>(margin = '600px') {
  const ref = useRef<T>(null);
  const [near, setNear] = useState(typeof IntersectionObserver === 'undefined');
  useEffect(() => {
    const el = ref.current;
    if (near || !el) return;
    const io = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) setNear(true);
      },
      { rootMargin: margin },
    );
    io.observe(el);
    return () => io.disconnect();
  }, [near, margin]);
  return [ref, near] as const;
}
