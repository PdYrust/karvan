import type { ReactNode } from 'react';

/** The page's one h1, with an optional line under it. */
export function PageHeader({ title, description }: { title: string; description?: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <h1 className="text-title font-bold text-kumo-strong">{title}</h1>
      {description ? <p className="text-kumo-subtle">{description}</p> : null}
    </div>
  );
}
