import type { ReactNode } from 'react';
import { useId } from 'react';
import { useT } from '../../i18n/locale';
import { useFormat } from '../../lib/preferences/region';
import { certificatesMessages } from './messages';

/**
 * A section of the page as a Migrations-style list: the heading with its count,
 * the toolbar, the table (or one line when nothing matches), then how many show.
 */
export function ListFrame({
  title,
  total,
  shown,
  noMatch,
  toolbar,
  children,
}: {
  title: string;
  total: number;
  shown: number;
  noMatch: string;
  toolbar: ReactNode;
  children: ReactNode;
}) {
  const t = useT(certificatesMessages);
  const fmt = useFormat();
  const id = useId();
  return (
    <section aria-labelledby={id} className="flex flex-col gap-3">
      <h2 id={id} className="text-heading font-semibold text-kumo-strong">
        {title}
        <span className="ms-2 font-normal text-kumo-subtle tabular">{fmt.number(total)}</span>
      </h2>
      {toolbar}
      {shown === 0 ? (
        <p className="rounded-lg bg-kumo-base px-4 py-8 text-center text-kumo-subtle ring ring-kumo-hairline">
          {noMatch}
        </p>
      ) : (
        children
      )}
      <p className="text-sm text-kumo-subtle" aria-live="polite">
        {t('showing', { shown, total })}
      </p>
    </section>
  );
}
