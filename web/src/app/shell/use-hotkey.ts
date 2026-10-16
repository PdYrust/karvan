import { useEffect } from 'react';

/**
 * Calls onPress for Ctrl+key (Cmd+key on macOS), even while typing in a
 * field. Matches the physical key (e.code, like "KeyK"), so the shortcut also
 * works with a Persian keyboard layout.
 */
export function useModHotkey(code: string, onPress: () => void) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && !e.altKey && e.code === code) {
        e.preventDefault();
        onPress();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [code, onPress]);
}
