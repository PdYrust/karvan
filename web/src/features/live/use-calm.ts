import { useSyncExternalStore } from 'react';

/** The Network Information API's Data Saver flag, where the browser has it. */
interface SaveData {
  saveData?: boolean;
  addEventListener?: (type: 'change', fn: () => void) => void;
  removeEventListener?: (type: 'change', fn: () => void) => void;
}

const query = '(prefers-reduced-motion: reduce)';
const connection = (): SaveData | undefined =>
  (navigator as Navigator & { connection?: SaveData }).connection;

function subscribe(onChange: () => void): () => void {
  const media = matchMedia(query);
  media.addEventListener('change', onChange);
  connection()?.addEventListener?.('change', onChange);
  return () => {
    media.removeEventListener('change', onChange);
    connection()?.removeEventListener?.('change', onChange);
  };
}

/** Whether the operator asked for less motion or the browser saves data, right now. */
export const calmNow = () => matchMedia(query).matches || connection()?.saveData === true;

/**
 * True when the operator asked for less motion, or the browser saves data:
 * the live view then shows a plain meter instead of moving dots.
 */
export function useCalm(): boolean {
  return useSyncExternalStore(subscribe, calmNow, () => true);
}
