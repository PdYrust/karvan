import { cn } from '@cloudflare/kumo/utils';
import { createContext, use, useId, type ReactNode } from 'react';
// A row's title sits under the card's heading; a card without one puts it right under the page's.
const Titled = createContext(false);

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
      <div className="flex flex-col">
        <Titled value={Boolean(title)}>{children}</Titled>
      </div>
    </section>
  );
}

/**
 * One setting: title and a line of description at the start, the control at
 * the end, as in the dashboard's settings pages. Under 640 a wide control
 * drops below the text; a switch (`inline`) stays at the end of the title
 * line and wraps under the title, before the description, when the title
 * needs the whole line.
 */
export function SettingRow({
  title,
  description,
  control,
  inline = false,
  badge,
  children,
  labelId,
}: {
  title: string;
  /** A status badge after the title, such as On or Off. */
  badge?: ReactNode;
  description?: ReactNode;
  control?: ReactNode;
  /** The control is small (a switch) and shares the title's line on phones. */
  inline?: boolean;
  /** More detail under the description, such as a warning. */
  children?: ReactNode;
  /** Id for the title, so the control can be labelled by it. */
  labelId?: string;
}) {
  const Heading = use(Titled) ? 'h3' : 'h2';
  const name = (
    <Heading id={labelId} className="text-base font-medium text-kumo-strong">
      {title}
    </Heading>
  );
  const heading = badge ? (
    <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
      {name}
      {badge}
    </div>
  ) : (
    name
  );
  const text = (
    <>
      {description ? <p className="max-w-prose text-sm text-kumo-subtle">{description}</p> : null}
      {children}
    </>
  );
  if (inline) {
    // Phones: one wrapping line of title and switch, the rest on full lines under it.
    // From 640 a grid: the text in the first column, the switch at the end of the row.
    return (
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-b border-kumo-hairline py-5 last:border-b-0 sm:grid sm:grid-cols-[minmax(0,1fr)_auto] sm:items-start sm:gap-x-8">
        {heading}
        <div className="flex shrink-0 items-center sm:col-start-2 sm:row-span-2 sm:row-start-1 sm:min-h-6">
          {control}
        </div>
        <div className="flex basis-full flex-col gap-1 sm:col-start-1">{text}</div>
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-3 border-b border-kumo-hairline py-5 last:border-b-0 sm:flex-row sm:items-start sm:justify-between sm:gap-8">
      <div className="flex min-w-0 flex-col gap-1">
        {heading}
        {text}
      </div>
      {control ? <div className="flex shrink-0 items-center sm:min-h-6">{control}</div> : null}
    </div>
  );
}
