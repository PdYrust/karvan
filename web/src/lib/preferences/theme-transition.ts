import { flushSync } from 'react-dom';
import { reducedMotion } from '../motion/reduced-motion';

type TransitionDoc = Document & { startViewTransition?: (update: () => void) => unknown };

/**
 * Runs a theme change as a crossfade between the old and the new page
 * (styles/motion.css times it). Where the browser has no View Transitions, or
 * the operator asked for less motion, the theme changes at once.
 */
export function withThemeTransition(
  apply: () => void,
  doc: TransitionDoc = document,
): 'faded' | 'instant' {
  if (typeof doc.startViewTransition !== 'function' || reducedMotion()) {
    apply();
    return 'instant';
  }
  // The new snapshot is taken when the callback returns, so React renders inside it.
  doc.startViewTransition(() => flushSync(apply));
  return 'faded';
}
