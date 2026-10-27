import { useEffect, useState } from 'react';
/** Twice the longest close (DESIGN.md section 5), so a slow phone that starts it late still ends it. */
export const closeMotion = 440;
/**
 * `value`, kept for `ms` after it goes away (turns null), so what shows it
 * (a dialog naming a backup, the two-step setup) keeps its content while
 * it closes with its motion, then lets it go.
 */
export function useLingering<T>(value: T | null, ms = closeMotion): T | null {
  const [kept, setKept] = useState<T | null>(value);
  if (value !== null && value !== kept) setKept(value);
  useEffect(() => {
    if (value !== null || kept === null) return;
    const id = setTimeout(() => setKept(null), ms);
    return () => clearTimeout(id);
  }, [value, kept, ms]);
  return value ?? kept;
}
