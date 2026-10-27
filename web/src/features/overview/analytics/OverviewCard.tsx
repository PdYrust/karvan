import { BarList } from '../../../components/charts/BarList';
import { CompactCard } from '../../../components/charts/CompactCard';
import { CompactChart } from '../../../components/charts/CompactChart';
import type { ServerView } from '../../analytics/server-view';
import type { ServerCard } from './server-cards';

/** One card of the overview's analytics; pressing it opens its detail when it has one. */
export function OverviewCard({
  card,
  view,
  within,
  onOpen,
}: {
  card: ServerCard;
  view: ServerView;
  within: string;
  onOpen: () => void;
}) {
  return (
    <CompactCard
      title={card.title}
      value={card.value}
      unit={card.unit}
      change={card.change}
      within={within}
      sub={card.sub}
      onOpen={card.detail ? onOpen : undefined}
      chart={
        card.chart ? (
          <CompactChart
            times={view.times}
            values={card.chart.values}
            domain={view.domain}
            step={view.step}
            spacing={view.spacing}
            max={card.chart.max}
            className="h-30"
          />
        ) : undefined
      }
    >
      {card.list ? <BarList label={card.title} items={card.list} /> : undefined}
    </CompactCard>
  );
}
