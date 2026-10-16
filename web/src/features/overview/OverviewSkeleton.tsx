import { cn } from '@cloudflare/kumo/utils';

function Bone({ className }: { className?: string }) {
  return <div className={cn('animate-pulse rounded-md bg-kumo-recessed', className)} />;
}

/** Same shape as the overview body, so nothing jumps when the data arrives. */
export function OverviewSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="grid lg:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="flex flex-col gap-6 p-4 sm:p-8">
        <div className="flex h-72 flex-col items-center justify-center gap-4 rounded-lg border border-kumo-hairline bg-kumo-base">
          <Bone className="size-11" />
          <Bone className="h-5 w-56" />
          <Bone className="h-3.5 w-80 max-w-[80%]" />
        </div>
      </div>
      <div className="flex flex-col gap-4 border-kumo-hairline p-4 sm:p-8 lg:border-s">
        {[0, 1, 2].map((i) => (
          <div key={i} className="flex flex-col gap-3 pb-4">
            <Bone className="h-4 w-28" />
            <Bone className="h-3.5 w-full" />
          </div>
        ))}
      </div>
    </div>
  );
}
