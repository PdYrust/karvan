import { useSyncExternalStore } from 'react';

// Tailwind's lg breakpoint, where the sidebar column shows.
const query = '(min-width: 64rem)';

function subscribe(onChange: () => void): () => void {
  const media = matchMedia(query);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}

/** Whether the screen is wide enough for the sidebar column; phones use the sheet instead. */
export function useWide(): boolean {
  return useSyncExternalStore(subscribe, () => matchMedia(query).matches);
}
