import type { PanelAnalytics, RangeKey } from '../../../lib/api/analytics';
import { databaseRange, rangeSpan } from '../range';

/**
 * Where a panel's charts end, their step and the time they cover, in ms.
 * They end at the panel's last reading rather than this browser's clock,
 * which may be off.
 */
export function panelWindow(a: Pick<PanelAnalytics, 'panel' | 'step'>, range: RangeKey) {
  const end = Date.parse(a.panel.at) || (a.panel.points.at(-1)?.t ?? 0) * 1000;
  const domain: [number, number] = [end - rangeSpan[databaseRange(range)], end];
  return { end, step: a.step * 1000, domain };
}
