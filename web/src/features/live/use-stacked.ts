import { useSyncExternalStore } from 'react';

// Under Tailwind's sm breakpoint, where the scene stands up.
const query = '(max-width: 39.98rem)';

function subscribe(onChange: () => void): () => void {
  const media = matchMedia(query);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}

/** Whether the screen is a phone's, where the live scene stands up. */
export function useStacked(): boolean {
  return useSyncExternalStore(subscribe, () => matchMedia(query).matches);
}
