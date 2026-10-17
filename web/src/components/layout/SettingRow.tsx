import { cn } from '@cloudflare/kumo/utils';
import { useId, type ReactNode } from 'react';

/** A card of settings, each row split from the next by a hairline. */
export function SettingsCard({
  title,
  children,
  className,
}: {
  title?: string;
  children: ReactNode;
  className?: string;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={title ? id : undefined}
      className={cn('rounded-lg bg-kumo-base px-4 ring ring-kumo-hairline sm:px-6', className)}
    >
      {title ? (
        <h2 id={id} className="pt-5 text-heading font-semibold text-kumo-strong">
          {title}
        </h2>
      ) : null}
      <div className="flex flex-col">{children}</div>
    </section>
  );
}

/**
 * One setting: title and a line of description at the start, the control at
 * the end, as in the dashboard's settings pages. Under 640 the control drops
 * below the text.
 */
export function SettingRow({
  title,
  description,
  control,
  children,
  labelId,
}: {
  title: string;
  description?: ReactNode;
  control?: ReactNode;
  /** More detail under the description, such as a warning. */
  children?: ReactNode;
  /** Id for the title, so the control can be labelled by it. */
  labelId?: string;
}) {
  return (
    <div className="flex flex-col gap-3 border-b border-kumo-hairline py-5 last:border-b-0 sm:flex-row sm:items-start sm:justify-between sm:gap-8">
      <div className="flex min-w-0 flex-col gap-1">
        <h3 id={labelId} className="text-base font-medium text-kumo-strong">
          {title}
        </h3>
        {description ? <p className="max-w-prose text-sm text-kumo-subtle">{description}</p> : null}
        {children}
      </div>
      {control ? <div className="flex shrink-0 items-center sm:min-h-6">{control}</div> : null}
    </div>
  );
}
