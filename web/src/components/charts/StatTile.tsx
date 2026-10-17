import type { ReactNode } from 'react';
import { Bone } from '../feedback/PageSkeleton';

/**
 * A number with its label, a quiet line under it and a sparkline (or a
 * meter) along the bottom edge. Tiles in one row share a height.
 */
export function StatTile({
  label,
  value,
  unit,
  sub,
  children,
}: {
  label: string;
  value: ReactNode;
  unit?: string;
  sub?: ReactNode;
  /** Drawn along the bottom edge, full width. */
  children?: ReactNode;
}) {
  return (
    <div className="flex min-h-32 min-w-0 flex-col overflow-hidden rounded-lg bg-kumo-base ring ring-kumo-hairline">
      <div className="flex min-w-0 flex-col gap-0.5 px-4 pt-3">
        <span className="truncate text-sm text-kumo-subtle">{label}</span>
        <span className="flex min-w-0 items-baseline gap-1">
          <span className="truncate text-title font-semibold text-kumo-strong tabular">
            {value}
          </span>
          {unit ? <span className="shrink-0 text-sm text-kumo-subtle">{unit}</span> : null}
        </span>
        {sub ? <span className="line-clamp-2 text-sm text-kumo-subtle">{sub}</span> : null}
      </div>
      {children ? <div className="mt-auto pt-2">{children}</div> : null}
    </div>
  );
}

/** A tile while its number loads, the same size as the real one. */
export function StatTileSkeleton() {
  return (
    <div className="flex min-h-32 flex-col gap-2 rounded-lg bg-kumo-base px-4 pt-3 ring ring-kumo-hairline">
      <Bone className="h-4 w-20" />
      <Bone className="h-7 w-28" />
      <Bone className="h-4 w-16" />
    </div>
  );
}
