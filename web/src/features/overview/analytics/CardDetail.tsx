import type { ReactNode } from 'react';
import { BarList } from '../../../components/charts/BarList';
import { Legend } from '../../../components/charts/Legend';
import { TimeSeries } from '../../../components/charts/TimeSeries';
import { SheetDialog } from '../../../components/dialogs/SheetDialog';
import { useT } from '../../../i18n/locale';
import { analyticsMessages } from '../../analytics/messages';
import type { ServerView } from '../../analytics/server-view';
import { overviewMessages } from '../messages';
import type { ServerCard } from './server-cards';

function Inner({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3 rounded-lg p-4 ring ring-kumo-hairline">
      <h3 className="text-sm font-medium text-kumo-default">{title}</h3>
      {children}
    </section>
  );
}

/** A card's detail: the series over time, then who leads. */
export function CardDetail({
  card,
  view,
  within,
  open,
  onOpenChange,
}: {
  card: ServerCard | undefined;
  view: ServerView | undefined;
  within: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT(overviewMessages);
  const ta = useT(analyticsMessages);
  const detail = card?.detail;
  return (
    <SheetDialog
      open={open && detail !== undefined}
      onOpenChange={onOpenChange}
      title={card?.title}
      description={t('detailLine')}
    >
      {detail && card && view ? (
        <>
          <Inner title={t('trendTitle')}>
            <Legend items={detail.series} more={detail.more} />
            <TimeSeries
              label={ta('chartLabel', { chart: card.title, range: within })}
              times={view.times}
              series={detail.series}
              domain={view.domain}
              step={view.step}
              spacing={view.spacing}
              kind={detail.kind}
              format={detail.format}
              empty={ta('noData', { range: within })}
              height={240}
            />
          </Inner>
          {detail.contributors.length > 0 ? (
            <Inner title={t('contributorsTitle')}>
              <BarList label={t('contributorsTitle')} items={detail.contributors} />
            </Inner>
          ) : null}
        </>
      ) : null}
    </SheetDialog>
  );
}
