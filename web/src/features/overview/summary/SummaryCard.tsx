import { useId, type ReactNode } from 'react';
import { Bone } from '../../../components/feedback/PageSkeleton';

/** A summary card: its title, maybe a badge, then whatever shape its numbers need. */
export function SummaryCard({
  title,
  badge,
  children,
}: {
  title: string;
  badge?: ReactNode;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className="flex min-w-0 flex-col gap-3 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline"
    >
      <div className="flex min-h-6 items-center justify-between gap-3">
        <h3 id={id} className="text-heading font-semibold text-kumo-strong">
          {title}
        </h3>
        {badge}
      </div>
      {children}
    </section>
  );
}

/** A card's body while its numbers load. */
export function SummarySkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col gap-3">
      <Bone className="h-8 w-24" />
      <Bone className="h-1.5 w-full" />
      <Bone className="h-4 w-40" />
    </div>
  );
}
