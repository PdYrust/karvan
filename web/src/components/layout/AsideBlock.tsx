import { useId, type ReactNode } from 'react';

/** One block of a page's aside: a heading, an optional status badge, a line or two, split by hairlines. */
export function AsideBlock({
  title,
  badge,
  children,
}: {
  title: string;
  badge?: ReactNode;
  children?: ReactNode;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className="flex flex-col gap-2 border-b border-kumo-hairline py-5 first:pt-0 last:border-b-0"
    >
      <div className="flex items-center justify-between gap-3">
        <h2 id={id} className="text-heading font-semibold text-kumo-strong">
          {title}
        </h2>
        {badge}
      </div>
      {children}
    </section>
  );
}
