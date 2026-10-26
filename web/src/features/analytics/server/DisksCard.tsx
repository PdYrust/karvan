import { Bar } from '../../../components/charts/Bar';
import { useT } from '../../../i18n/locale';
import type { Disk } from '../../../lib/api/analytics';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../ChartCard';
import { analyticsMessages } from '../messages';

/** Every real file system on the server, root first, with how full it is. */
export function DisksCard({ disks }: { disks: readonly Disk[] }) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  return (
    <ChartCard title={t('disksTitle')}>
      <ul className="flex flex-col gap-5">
        {disks.map((d) => (
          <li key={d.mount} className="flex flex-col gap-2">
            <span className="flex items-baseline justify-between gap-3">
              <span className="flex min-w-0 items-baseline gap-2">
                <bdi dir="ltr" className="truncate font-mono text-sm text-kumo-default">
                  {d.mount}
                </bdi>
                <bdi dir="ltr" className="text-sm text-kumo-subtle">
                  {d.type}
                </bdi>
              </span>
              <span className="shrink-0 text-sm font-medium text-kumo-strong tabular">
                {fmt.percent(d.used / (d.total || 1), 0)}
              </span>
            </span>
            <Bar grow share={d.used / (d.total || 1)} />
            <span className="flex justify-between gap-3 text-sm text-kumo-subtle tabular">
              <span>{t('diskOf', { used: fmt.bytes(d.used), total: fmt.bytes(d.total) })}</span>
              <span>{t('diskFree', { free: fmt.bytes(d.free) })}</span>
            </span>
          </li>
        ))}
      </ul>
    </ChartCard>
  );
}
