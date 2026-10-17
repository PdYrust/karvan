import { useSearchParams } from 'react-router';
import { rangeKeys, type RangeKey } from '../../lib/api/analytics';

const minute = 60_000;
const day = 24 * 60 * minute;

/** How far back each range looks, in ms. Live is the 15 minutes the server keeps in memory. */
export const rangeSpan: Readonly<Record<RangeKey, number>> = {
  live: 15 * minute,
  '1h': 60 * minute,
  '24h': day,
  '7d': 7 * day,
  '30d': 30 * day,
};

/** Databases are read once a minute, so their live view is the last hour. */
export function databaseRange(r: RangeKey): Exclude<RangeKey, 'live'> {
  return r === 'live' ? '1h' : r;
}

export function isRange(v: string | null): v is RangeKey {
  return v !== null && (rangeKeys as readonly string[]).includes(v);
}

/** The range lives in the address (?range=24h), so a link or a reload keeps it. Live by default. */
export function useRange(): [RangeKey, (r: RangeKey) => void] {
  const [params, setParams] = useSearchParams();
  const raw = params.get('range');
  const range = isRange(raw) ? raw : 'live';
  const set = (r: RangeKey) =>
    setParams(
      (p) => {
        const next = new URLSearchParams(p);
        if (r === 'live') next.delete('range');
        else next.set('range', r);
        return next;
      },
      { replace: true },
    );
  return [range, set];
}
