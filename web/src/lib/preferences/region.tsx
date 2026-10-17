import { createContext, use, useMemo, useState, type ReactNode } from 'react';
import { useLocale } from '../../i18n/locale';
import { makeFormat, type Format } from '../format/format';
import {
  readCalendarPref,
  readTimeZone,
  saveCalendarPref,
  saveTimeZone,
  type CalendarPref,
} from './preferences';

interface RegionValue {
  timeZone: string;
  calendarPref: CalendarPref;
  setTimeZone: (tz: string) => void;
  setCalendarPref: (p: CalendarPref) => void;
}

const RegionContext = createContext<RegionValue | null>(null);

/** Time zone and calendar, remembered per browser like language and theme. */
export function RegionProvider({ children }: { children: ReactNode }) {
  const [timeZone, setZone] = useState(readTimeZone);
  const [calendarPref, setCalendar] = useState(readCalendarPref);
  const value = useMemo<RegionValue>(
    () => ({
      timeZone,
      calendarPref,
      setTimeZone: (tz) => {
        saveTimeZone(tz);
        setZone(tz);
      },
      setCalendarPref: (p) => {
        saveCalendarPref(p);
        setCalendar(p);
      },
    }),
    [timeZone, calendarPref],
  );
  return <RegionContext value={value}>{children}</RegionContext>;
}

export function useRegion(): RegionValue {
  const ctx = use(RegionContext);
  if (!ctx) throw new Error('useRegion needs a RegionProvider');
  return ctx;
}

/** Formatters for the operator's language, zone and calendar. */
export function useFormat(): Format {
  const { locale } = useLocale();
  const { timeZone, calendarPref } = useRegion();
  return useMemo(
    () => makeFormat(locale, { timeZone, calendar: calendarPref }),
    [locale, timeZone, calendarPref],
  );
}
