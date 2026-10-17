import { useT } from '../../../i18n/locale';
import type { IfaceRate } from '../../../lib/api/analytics';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../ChartCard';
import { analyticsMessages } from '../messages';

/** Each physical network card: its rate now and what it moved since the server booted. */
export function InterfacesCard({ ifaces }: { ifaces: readonly IfaceRate[] }) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const cell = 'py-2.5 text-end tabular';
  return (
    <ChartCard title={t('interfaces')}>
      <table className="w-full text-sm">
        <thead className="text-kumo-subtle">
          <tr className="border-b border-kumo-hairline">
            <th scope="col" className="pb-2 text-start font-normal">
              {t('ifaceName')}
            </th>
            <th scope="col" className="pb-2 text-end font-normal">
              {t('seriesIn')}
            </th>
            <th scope="col" className="pb-2 text-end font-normal">
              {t('seriesOut')}
            </th>
          </tr>
        </thead>
        <tbody>
          {ifaces.map((i) => (
            <tr key={i.name} className="border-b border-kumo-hairline last:border-0">
              <th scope="row" className="py-2.5 text-start font-normal">
                <bdi dir="ltr" className="font-mono text-kumo-default">
                  {i.name}
                </bdi>
              </th>
              <td className={cell}>
                <span className="block font-medium text-kumo-strong">{fmt.rate(i.rx)}</span>
                <span className="block text-kumo-subtle">
                  {t('ifaceReceived')} {fmt.bytes(i.totalRx)}
                </span>
              </td>
              <td className={cell}>
                <span className="block font-medium text-kumo-strong">{fmt.rate(i.tx)}</span>
                <span className="block text-kumo-subtle">
                  {t('ifaceSent')} {fmt.bytes(i.totalTx)}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </ChartCard>
  );
}
