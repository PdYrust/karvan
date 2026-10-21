import { useOutletContext } from 'react-router';

/** What the shell hands the page it shows. */
export interface ShellOutlet {
  /** Opens the Ctrl K palette. */
  openSearch: () => void;
}

export function useShell(): ShellOutlet {
  return useOutletContext<ShellOutlet>();
}
