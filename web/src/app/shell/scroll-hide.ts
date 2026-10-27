import { useCallback, useEffect, useRef, useState } from 'react';
export interface ScrollHide {
  hidden: boolean;
  /** Where the last change of direction was counted from. */
  anchor: number;
}
/**
 * The bar's next state for a scroll position: shown near the top, hidden once
 * the page moves down past `after`, shown again on any move up. A move under
 * `tolerance` px (a jitter, a bounce) changes nothing.
 */
export function nextScrollHide(
  s: ScrollHide,
  y: number,
  { after, tolerance = 6 }: { after: number; tolerance?: number },
): ScrollHide {
  if (y <= after) return s.hidden || s.anchor !== y ? { hidden: false, anchor: y } : s;
  const moved = y - s.anchor;
  if (Math.abs(moved) < tolerance) return s;
  return { hidden: moved > 0, anchor: y };
}
const popups = '[role="dialog"], [role="menu"], [role="listbox"], dialog[open]';
/**
 * Hides a top bar while the page scrolls down and brings it back on the way up,
 * like a phone browser's address bar. Off, or while `hold` (a sheet, menu or
 * dialog is open), it stays shown. `show` brings it back, for focus moving in.
 */
export function useScrollHide({
  enabled,
  hold,
  after,
}: {
  enabled: boolean;
  hold: boolean;
  after: number;
}) {
  const [hidden, setHidden] = useState(false);
  const state = useRef<ScrollHide>({ hidden: false, anchor: 0 });
  useEffect(() => {
    if (!enabled) return;
    state.current = { hidden: false, anchor: window.scrollY };
    const read = () => {
      // A menu, select or dialog the page opened keeps the bar where it is.
      const next = document.querySelector(popups)
        ? { hidden: false, anchor: window.scrollY }
        : nextScrollHide(state.current, window.scrollY, { after });
      state.current = next;
      setHidden(next.hidden);
    };
    window.addEventListener('scroll', read, { passive: true });
    return () => window.removeEventListener('scroll', read);
  }, [enabled, after]);
  const show = useCallback(() => {
    state.current = { hidden: false, anchor: window.scrollY };
    setHidden(false);
  }, []);
  return { hidden: enabled && !hold && hidden, show };
}
