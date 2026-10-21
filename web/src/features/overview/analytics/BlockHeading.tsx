import { CaretRightIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { StatTileSkeleton } from '../../../components/charts/StatTile';

/** The name of one block of numbers, linking to its full analytics. */
export function BlockHeading({
  id,
  to,
  children,
}: {
  id: string;
  to: string;
  children: ReactNode;
}) {
  return (
    <h3 id={id} className="flex min-h-8 min-w-0 items-center">
      <Link
        to={to}
        className="inline-flex min-w-0 items-center gap-2 rounded-sm font-medium text-kumo-strong hover:underline pointer-coarse:min-h-11"
      >
        {children}
        <CaretRightIcon
          size={14}
          aria-hidden="true"
          className="shrink-0 text-kumo-subtle rtl:-scale-x-100"
        />
      </Link>
    </h3>
  );
}

/** A row of tiles while their numbers load. */
export function TilesSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {[0, 1, 2, 3].map((i) => (
        <StatTileSkeleton key={i} />
      ))}
    </div>
  );
}
