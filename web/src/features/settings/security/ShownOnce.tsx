import { Button } from '@cloudflare/kumo/components/button';
import { useId, type ReactNode } from 'react';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { useT } from '../../../i18n/locale';
import { CopyValue } from './CopyValue';
import { securityMessages } from './messages';

/**
 * Values Karvan shows once: each with Copy, a warning, and "I saved
 * them" as the only way to close the card. They live in the caller's state.
 */
export function ShownOnce({
  title,
  values,
  children,
  onDone,
}: {
  title: string;
  values: readonly { label: string; value: string }[];
  /** What else to know, such as that other sessions ended. */
  children?: ReactNode;
  onDone: () => void;
}) {
  const t = useT(securityMessages);
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className="flex flex-col gap-4 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-6"
    >
      <h2 id={id} className="text-heading font-semibold text-kumo-strong">
        {title}
      </h2>
      <InlineAlert tone="warning">{t('shownOnce')}</InlineAlert>
      {children}
      <div className="flex flex-col gap-4">
        {values.map((v) => (
          <CopyValue key={v.label} label={v.label} value={v.value} />
        ))}
      </div>
      <div className="flex justify-end">
        <Button variant="primary" className="pointer-coarse:h-9" onClick={onDone}>
          {t('savedThem')}
        </Button>
      </div>
    </section>
  );
}
