import { createContext, use, useEffect, useMemo, useState, type ReactNode } from 'react';
import {
  applyTheme,
  readThemePref,
  resolveMode,
  saveThemePref,
  watchSystemTheme,
  type Mode,
  type ThemePref,
} from './preferences';

interface ThemeContextValue {
  pref: ThemePref;
  mode: Mode;
  setPref: (p: ThemePref) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [pref, setPrefState] = useState<ThemePref>(readThemePref);
  const [mode, setMode] = useState<Mode>(() => resolveMode(pref));

  useEffect(() => {
    if (pref !== 'system') return;
    return watchSystemTheme(() => setMode(applyTheme('system')));
  }, [pref]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      pref,
      mode,
      setPref: (p) => {
        setPrefState(p);
        setMode(saveThemePref(p));
      },
    }),
    [pref, mode],
  );
  return <ThemeContext value={value}>{children}</ThemeContext>;
}

export function useTheme(): ThemeContextValue {
  const ctx = use(ThemeContext);
  if (!ctx) throw new Error('useTheme needs a ThemeProvider');
  return ctx;
}
