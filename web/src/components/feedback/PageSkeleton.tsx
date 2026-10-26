import { cn } from '@cloudflare/kumo/utils';

export function Bone({ className }: { className?: string }) {
  return <div className={cn('k-bone rounded-md bg-kumo-recessed', className)} />;
}

/** While a page's code loads: a header and two blocks, the shape every page starts with. */
export function PageSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col">
      <div className="flex flex-col gap-3 border-b border-kumo-hairline px-4 py-6 sm:px-8 sm:py-8">
        <Bone className="h-6 w-44" />
        <Bone className="h-4 w-72 max-w-[80%]" />
      </div>
      <div className="flex flex-col gap-4 p-4 sm:p-8">
        <Bone className="h-28 w-full" />
        <Bone className="h-56 w-full" />
      </div>
    </div>
  );
}
