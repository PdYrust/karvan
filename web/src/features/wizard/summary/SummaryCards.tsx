import { BarList, type BarItem } from '../../../components/charts/BarList';
import { useT } from '../../../i18n/locale';
import type { JobSummary } from '../../../lib/api/migrations';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../../analytics/ChartCard';
import { protocolName } from '../protocols';
import { summaryMessages, type SummaryKey } from './messages';

/** Statuses both panels use, in the order an operator reads them. */
const statuses = ['active', 'on_hold', 'limited', 'expired', 'disabled'] as const;
const expiries = ['expired', 'week', 'month', 'later', 'onHold', 'never'] as const;

function useItems() {
  const fmt = useFormat();
  return (entries: readonly [key: string, label: BarItem['label'], n: number][]): BarItem[] =>
    entries.map(([key, label, n]) => ({ key, label, value: n, display: fmt.number(n) }));
}

/** The old panel's users three ways: by status, by when they expire, by protocol. */
export function SummaryCards({ summary }: { summary: JobSummary }) {
  const t = useT(summaryMessages);
  const fmt = useFormat();
  const items = useItems();
  const byStatus = summary.statuses ?? {};
  const known: readonly string[] = statuses;
  const others = Object.keys(byStatus).filter((s) => !known.includes(s));
  const status = items(
    [...statuses, ...others].map((s) => [
      s,
      known.includes(s) ? t(`status_${s}` as SummaryKey) : <bdi dir="ltr">{s}</bdi>,
      byStatus[s as keyof typeof byStatus] ?? 0,
    ]),
  );
  const expiry = items(expiries.map((e) => [e, t(`exp_${e}`), summary.expiries[e]]));
  const protocols = items(
    Object.entries(summary.protocols)
      .sort(([a, x], [b, y]) => y - x || a.localeCompare(b))
      .map(([p, n]) => [p, <bdi dir="ltr">{protocolName(p)}</bdi>, n]),
  );
  const empty = summary.users === 0;
  return (
    <div className="grid gap-4 lg:grid-cols-3">
      <ChartCard title={t('byStatus')} aside={fmt.number(summary.users)}>
        {empty ? <Empty text={t('noUsers')} /> : <BarList label={t('byStatus')} items={status} />}
      </ChartCard>
      <ChartCard title={t('expiry')}>
        {empty ? <Empty text={t('noUsers')} /> : <BarList label={t('expiry')} items={expiry} />}
      </ChartCard>
      <ChartCard title={t('protocols')}>
        {protocols.length ? (
          <BarList label={t('protocols')} items={protocols} />
        ) : (
          <Empty text={t(empty ? 'noUsers' : 'noProtocols')} />
        )}
      </ChartCard>
    </div>
  );
}

function Empty({ text }: { text: string }) {
  return <p className="text-sm text-kumo-subtle">{text}</p>;
}
