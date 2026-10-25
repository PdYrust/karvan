import { useEffect, useState } from 'react';

/** Seconds until a time, ticking every second; null without one. */
export function useSecondsLeft(until: string | undefined): number | null {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!until) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [until]);
  if (!until) return null;
  return Math.max(0, Math.round((Date.parse(until) - now) / 1000));
}
