import { LayerCard } from '@cloudflare/kumo/components/layer-card';
import { createContext, use, useId, type ReactNode } from 'react';

type Level = 2 | 3 | 4;

const HeadingLevel = createContext<Level>(2);

/** Cards inside a block that has its own heading take the level under it (the overview's panel blocks). */
export function CardHeadings({ level, children }: { level: Level; children: ReactNode }) {
  return <HeadingLevel value={level}>{children}</HeadingLevel>;
}

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
  const Heading = `h${String(use(HeadingLevel))}` as 'h2' | 'h3' | 'h4';
  return (
    <LayerCard render={<section aria-labelledby={id} />} className={className}>
      <LayerCard.Secondary className="justify-between gap-3">
        <Heading id={id} className="truncate">
          {title}
        </Heading>
        {aside ? <span className="shrink-0 text-sm font-normal">{aside}</span> : null}
      </LayerCard.Secondary>
      <LayerCard.Primary className="flex-1 gap-4 p-4">{children}</LayerCard.Primary>
    </LayerCard>
  );
}
