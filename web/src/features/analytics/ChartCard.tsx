import { LayerCard } from '@cloudflare/kumo/components/layer-card';
import { useId, type ReactNode } from 'react';

/** One analytics block: Kumo's layer card with a tinted title row over a white body. */
export function ChartCard({
  title,
  aside,
  className,
  children,
}: {
  title: string;
  /** Quiet text or a control at the end of the title row. */
  aside?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <LayerCard render={<section aria-labelledby={id} />} className={className}>
      <LayerCard.Secondary className="justify-between gap-3">
        <h2 id={id} className="truncate">
          {title}
        </h2>
        {aside ? <span className="shrink-0 text-sm font-normal">{aside}</span> : null}
      </LayerCard.Secondary>
      <LayerCard.Primary className="flex-1 gap-4 p-4">{children}</LayerCard.Primary>
    </LayerCard>
  );
}
