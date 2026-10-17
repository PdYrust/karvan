import { defaultTimeZone } from '../../lib/format/date';

/** Zones operators of Iranian panels most often work in; the current one is always offered. */
const common = [
  defaultTimeZone,
  'UTC',
  'Asia/Dubai',
  'Europe/Istanbul',
  'Europe/Moscow',
  'Europe/Berlin',
  'Europe/Amsterdam',
  'Europe/London',
  'America/New_York',
  'America/Los_Angeles',
  'Asia/Tokyo',
];

function offsetOf(id: string): string {
  const part = new Intl.DateTimeFormat('en', { timeZone: id, timeZoneName: 'shortOffset' })
    .formatToParts(new Date())
    .find((p) => p.type === 'timeZoneName');
  return part?.value ?? id;
}

export function zoneChoices(current: string): { id: string; offset: string }[] {
  const ids = common.includes(current) ? common : [current, ...common];
  return ids.map((id) => ({ id, offset: offsetOf(id) }));
}
