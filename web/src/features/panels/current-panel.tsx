import { createContext, use, useMemo, useState, type ReactNode } from 'react';
import type { Panel } from '../../lib/api/panels';
import { readCurrentPanel, saveCurrentPanel } from '../../lib/preferences/preferences';
import { usePanels } from './use-panels';

interface CurrentPanelValue {
  /** The chosen panel, or null for "All panels" (also when the saved one is gone). */
  panel: Panel | null;
  /**
   * False until the panel list has loaded or failed. A list filtered by the
   * current panel waits for it, so it does not first load every panel's rows.
   */
  settled: boolean;
  select: (id: number | null) => void;
}

const CurrentPanelContext = createContext<CurrentPanelValue | null>(null);

/**
 * The panel the sidebar switcher points at: the wizard's default
 * destination and a filter on lists. Remembered per browser.
 */
export function CurrentPanelProvider({ children }: { children: ReactNode }) {
  const panels = usePanels();
  const [id, setId] = useState(readCurrentPanel);
  const value = useMemo<CurrentPanelValue>(
    () => ({
      panel: panels.data?.find((p) => p.id === id) ?? null,
      settled: !panels.isPending,
      select: (next) => {
        saveCurrentPanel(next);
        setId(next);
      },
    }),
    [panels.data, panels.isPending, id],
  );
  return <CurrentPanelContext value={value}>{children}</CurrentPanelContext>;
}

export function useCurrentPanel(): CurrentPanelValue {
  const ctx = use(CurrentPanelContext);
  if (!ctx) throw new Error('useCurrentPanel needs a CurrentPanelProvider');
  return ctx;
}
