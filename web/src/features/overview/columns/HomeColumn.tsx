import { CaretRightIcon } from '@phosphor-icons/react';
import { useId, type ReactNode } from 'react';
import { Link } from 'react-router';
import { Bone } from '../../../components/feedback/PageSkeleton';

const chevron = 'shrink-0 text-kumo-subtle rtl:-scale-x-100';

/** One of the overview's three columns: a quiet title, a link when its page exists, then short rows. */
export function HomeColumn({
  title,
  to,
  children,
}: {
  title: string;
  to?: string;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <section aria-labelledby={id} className="flex min-w-0 flex-col gap-1">
      <h2 id={id} className="flex min-h-8 items-center text-sm font-medium text-kumo-subtle">
        {to ? (
          <Link
            to={to}
            className="inline-flex items-center gap-1 rounded-sm hover:text-kumo-default pointer-coarse:min-h-11"
          >
            {title}
            <CaretRightIcon size={12} aria-hidden="true" className={chevron} />
          </Link>
        ) : (
          title
        )}
      </h2>
      {children}
    </section>
  );
}

/** The rows of a column, split by hairlines. */
export function HomeRows({ children }: { children: ReactNode }) {
  return <ul className="flex flex-col">{children}</ul>;
}

/** A row that opens its page, with a chevron at the end; without `to` it only reads. */
export function HomeRow({ to, children }: { to?: string; children: ReactNode }) {
  const body = <span className="flex min-w-0 flex-1 items-center gap-3">{children}</span>;
  return (
    <li className="border-b border-kumo-hairline last:border-b-0">
      {to ? (
        // The tint reaches a little past the text, so the words keep the column's edge.
        <Link
          to={to}
          className="-mx-2 flex min-h-14 items-center gap-3 rounded-md px-2 py-2 hover:bg-kumo-tint"
        >
          {body}
          <CaretRightIcon size={14} aria-hidden="true" className={chevron} />
        </Link>
      ) : (
        <div className="flex min-h-14 items-center py-2">{body}</div>
      )}
    </li>
  );
}

/** What a column says when it has no rows, on a tinted box. */
export function HomeNote({ icon, children }: { icon?: ReactNode; children: ReactNode }) {
  return (
    <p className="flex min-h-14 items-center justify-center gap-2 rounded-lg bg-kumo-tint px-4 py-3 text-center text-sm text-kumo-default">
      {icon ? <span className="shrink-0 text-kumo-subtle">{icon}</span> : null}
      <span>{children}</span>
    </p>
  );
}

/** A column's rows while they load. */
export function HomeRowsSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col">
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="flex min-h-14 items-center gap-3 border-b border-kumo-hairline last:border-b-0"
        >
          <Bone className="size-5" />
          <Bone className="h-4 w-40 max-w-[60%]" />
        </div>
      ))}
    </div>
  );
}
