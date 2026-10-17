import { useEffect, useRef } from 'react';
import { useLocation } from 'react-router';
import { getSession } from '../../lib/api/session';

/**
 * Moving to another page, tab or range is activity, even on a page that
 * only polls (polls leave the session's idle timer alone). Each move after
 * the first load asks about the session once, which resets the timer.
 */
export function useActivity() {
  const { pathname, search } = useLocation();
  const first = useRef(true);
  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    void getSession().catch(() => undefined);
  }, [pathname, search]);
}
