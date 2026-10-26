import type { ReactNode } from 'react';

/**
 * Wraps content that replaces a skeleton (`useFirstLoad`): with `on` it fades
 * in. The wrapper has no box of its own, so it leaves the layout alone.
 */
export function Reveal({ on, children }: { on: boolean; children: ReactNode }) {
  return <div className={on ? 'k-reveal' : 'contents'}>{children}</div>;
}
