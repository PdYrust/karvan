import { cn } from '@cloudflare/kumo/utils';
import { useId, type ReactNode } from 'react';

/** A titled region of a page: hairline border, no shadow (DESIGN.md section 4). */
export function Section({
  title,
  actions,
  className,
  children,
}: {
  title: string;
  actions?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className={cn('rounded-xl border border-kumo-hairline bg-kumo-base', className)}
    >
      <header className="flex min-h-12 items-center justify-between gap-3 border-b border-kumo-hairline px-4 sm:px-5">
        <h2 id={id} className="text-heading font-semibold text-kumo-strong">
          {title}
        </h2>
        {actions}
      </header>
      {children}
    </section>
  );
}
