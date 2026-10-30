import { useId, type ReactNode } from 'react';

/**
 * Nothing to show yet: an icon, what is missing, why, and one action. Kumo's
 * Empty in shape, with the card radius and heading size of the other cards.
 */
export function EmptyState({
  icon,
  title,
  children,
  action,
}: {
  icon: ReactNode;
  title: string;
  /** Why it is empty, in a line or two. */
  children?: ReactNode;
  action?: ReactNode;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className="flex flex-col items-center gap-3 rounded-lg bg-kumo-base px-6 py-12 text-center ring ring-kumo-hairline sm:py-16"
    >
      <span className="text-kumo-default">{icon}</span>
      <h2 id={id} className="mt-1 text-heading font-semibold text-kumo-strong">
        {title}
      </h2>
      {children ? <div className="max-w-md text-kumo-subtle">{children}</div> : null}
      {action ? <div className="mt-2 flex flex-wrap justify-center gap-2">{action}</div> : null}
    </section>
  );
}
