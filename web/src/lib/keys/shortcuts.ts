/** What a key press asks for, outside text fields. */
export type ShortcutAction =
  { type: 'help' } | { type: 'page-search' } | { type: 'goto'; to: string } | null;

/** How long "g" waits for the page letter. */
export const sequenceWindowMs = 1500;

/**
 * Reads `?`, `/` and `g` sequences. Keys are physical codes ("KeyO",
 * "Slash"), so the shortcuts work the same with a Persian layout.
 */
export function shortcutReader(targets: Readonly<Record<string, string>>) {
  let armedAt = -Infinity;
  return (code: string, shift: boolean, now: number): ShortcutAction => {
    const armed = now - armedAt < sequenceWindowMs;
    armedAt = -Infinity;
    if (code === 'Slash') return shift ? { type: 'help' } : { type: 'page-search' };
    if (armed) {
      const to = targets[code];
      return to ? { type: 'goto', to } : null;
    }
    if (code === 'KeyG' && !shift) armedAt = now;
    return null;
  };
}

/** True while the operator types into a field, where single keys must stay letters. */
export function isTyping(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  if (target instanceof HTMLTextAreaElement || target instanceof HTMLSelectElement) return true;
  if (target instanceof HTMLInputElement) {
    return !['checkbox', 'radio', 'button', 'submit', 'reset', 'range', 'color', 'file'].includes(
      target.type,
    );
  }
  return target.getAttribute('role') === 'combobox';
}
