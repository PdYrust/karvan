import { useEffect, useState } from 'react';

/** Seconds left until `until` (ms epoch), ticking once a second; calls onDone at zero. */
export function useCountdown(until: number | null, onDone: () => void): number {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (until === null) return;
    const tick = () => {
      const t = Date.now();
      setNow(t);
      if (t >= until) {
        window.clearInterval(id);
        onDone();
      }
    };
    const first = window.setTimeout(tick, 0);
    const id = window.setInterval(tick, 1000);
    return () => {
      window.clearTimeout(first);
      window.clearInterval(id);
    };
  }, [until, onDone]);

  return until === null ? 0 : Math.max(0, (until - now) / 1000);
}
