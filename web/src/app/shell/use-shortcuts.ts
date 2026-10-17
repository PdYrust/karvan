import { useEffect } from 'react';
import { useNavigate } from 'react-router';
import { isTyping, shortcutReader } from '../../lib/keys/shortcuts';
import { gotoTargets } from './routes-built';

/** `?` for the shortcut sheet, `/` for the page's search field, `g` then a letter for a page. */
export function useShortcuts(onHelp: () => void) {
  const navigate = useNavigate();
  useEffect(() => {
    const read = shortcutReader(gotoTargets());
    const handler = (e: KeyboardEvent) => {
      if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey || e.repeat) return;
      if (isTyping(e.target)) return;
      const action = read(e.code, e.shiftKey, performance.now());
      if (!action) return;
      if (action.type === 'help') {
        e.preventDefault();
        onHelp();
      } else if (action.type === 'goto') {
        e.preventDefault();
        void navigate(action.to);
      } else {
        const field = document.querySelector<HTMLElement>('[data-page-search]');
        if (field) {
          e.preventDefault();
          field.focus();
        }
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [navigate, onHelp]);
}
