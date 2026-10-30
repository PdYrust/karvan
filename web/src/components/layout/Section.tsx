import { cn } from '@cloudflare/kumo/utils';
import { useId, type ReactNode } from 'react';

/** A titled white card with a hairline border and no shadow. */
export function Section({
  title,
  actions,
  className,
  children,
}: {
  title?: ReactNode;
  actions?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={title ? id : undefined}
      className={cn(
        'overflow-hidden rounded-lg border border-kumo-hairline bg-kumo-base',
        className,
      )}
    >
      {title ? (
        <header className="flex min-h-12 items-center justify-between gap-3 px-4 pt-2">
          <h2 id={id} className="text-heading font-semibold text-kumo-strong">
            {title}
          </h2>
          {actions}
        </header>
      ) : null}
      {children}
    </section>
  );
}
