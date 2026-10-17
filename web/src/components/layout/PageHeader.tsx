import type { ReactNode } from 'react';

/**
 * The top of every page: an optional small label above the title (the
 * dashboard's breadcrumb), one h1, a line under it, actions at the end, and
 * tabs along the bottom hairline.
 */
export function PageHeader({
  title,
  description,
  actions,
  eyebrow,
  tabs,
}: {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  eyebrow?: ReactNode;
  tabs?: ReactNode;
}) {
  return (
    <div className="border-b border-kumo-hairline">
      <div className="flex flex-col gap-4 px-4 py-6 sm:flex-row sm:items-start sm:justify-between sm:px-8 sm:py-8">
        <div className="flex min-w-0 flex-col gap-1">
          {eyebrow ? <div className="text-sm text-kumo-subtle">{eyebrow}</div> : null}
          <h1 className="text-title font-semibold text-kumo-strong">{title}</h1>
          {description ? <p className="text-kumo-subtle">{description}</p> : null}
        </div>
        {actions ? (
          <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>
        ) : null}
      </div>
      {tabs ? <div className="-mt-2 px-2 sm:px-6">{tabs}</div> : null}
    </div>
  );
}
