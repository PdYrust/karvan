import type { ReactNode } from 'react';

/** The top of every page: one h1, a line under it, and the page's actions at the end. */
export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-4 border-b border-kumo-hairline px-4 py-6 sm:flex-row sm:items-start sm:justify-between sm:px-8 sm:py-8">
      <div className="flex flex-col gap-1">
        <h1 className="text-title font-semibold text-kumo-strong">{title}</h1>
        {description ? <p className="text-kumo-subtle">{description}</p> : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}
