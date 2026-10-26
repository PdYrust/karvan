import { useSyncExternalStore } from 'react';

const query = '(prefers-reduced-motion: reduce)';

/** Whether the operator asked for less motion, right now. */
export const reducedMotion = (): boolean =>
  typeof matchMedia === 'function' && matchMedia(query).matches;

function subscribe(onChange: () => void): () => void {
  if (typeof matchMedia !== 'function') return () => undefined;
  const media = matchMedia(query);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}

export function useReducedMotion(): boolean {
  return useSyncExternalStore(subscribe, reducedMotion, () => false);
}
