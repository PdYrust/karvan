import { useEffect } from 'react';
import { useLocation } from 'react-router';
import { recordRecent } from '../../lib/preferences/recents';
import { recentPath } from './recent-pages';

/** Remembers each page the operator opens, for the overview's Recents. */
export function useRecordRecents() {
  const { pathname } = useLocation();
  useEffect(() => {
    const path = recentPath(pathname);
    if (path) recordRecent(path);
  }, [pathname]);
}
