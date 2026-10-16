import { cn } from '@cloudflare/kumo/utils';

function Bone({ className }: { className?: string }) {
  return <div className={cn('animate-pulse rounded-md bg-kumo-recessed', className)} />;
}

function Block({ className, rows }: { className?: string; rows: number }) {
  return (
    <div className={cn('rounded-xl border border-kumo-hairline bg-kumo-base', className)}>
      <div className="border-b border-kumo-hairline px-5 py-4">
        <Bone className="h-4 w-28" />
      </div>
      <div className="flex flex-col gap-3 px-5 py-5">
        {Array.from({ length: rows }, (_, i) => (
          <Bone key={i} className={cn('h-3.5', i % 2 ? 'w-3/5' : 'w-11/12')} />
        ))}
      </div>
    </div>
  );
}

/** Same shape as the overview, so nothing jumps when the data arrives. */
export function OverviewSkeleton({ label }: { label: string }) {
  return (
    <div
      role="status"
      aria-label={label}
      className="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]"
    >
      <Block rows={5} className="lg:row-span-2" />
      <Block rows={2} />
      <Block rows={3} />
      <Block rows={4} className="lg:col-span-2" />
    </div>
  );
}
