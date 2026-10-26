import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { XIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { useT } from '../../i18n/locale';
import { dialogMessages } from './messages';

// On phones the dialog is a bottom sheet: full width, rounded on top, sliding up.
const sheet =
  'p-0 max-sm:inset-x-0 max-sm:top-auto max-sm:bottom-0 max-sm:max-w-none max-sm:translate-x-0 max-sm:rounded-b-none max-sm:rounded-t-2xl max-sm:duration-[220ms] max-sm:data-starting-style:translate-y-full max-sm:data-starting-style:scale-100 max-sm:data-ending-style:translate-y-full max-sm:data-ending-style:scale-100';

/**
 * A wide dialog on desktop and a bottom sheet on phones, scrolling inside:
 * a card's detail, a full list. Always rendered, so it opens and closes
 * with its motion.
 */
export function SheetDialog({
  open,
  onOpenChange,
  title,
  description,
  top,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: ReactNode;
  description?: string;
  /** Controls that stay above the scrolling body (a search, a sort). */
  top?: ReactNode;
  children: ReactNode;
}) {
  const t = useT(dialogMessages);
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog size="xl" className={sheet}>
        <div className="flex max-h-[calc(100dvh-8rem)] flex-col max-sm:max-h-[88dvh]">
          <div className="flex items-start justify-between gap-4 px-6 pt-5 pb-1 max-sm:px-4">
            <Dialog.Title className="text-heading font-semibold">{title}</Dialog.Title>
            <Dialog.Close
              render={(props) => (
                <Button
                  {...props}
                  variant="ghost"
                  shape="square"
                  size="sm"
                  className="-me-2 pointer-coarse:size-11"
                  aria-label={t('close')}
                  icon={<XIcon size={18} />}
                />
              )}
            />
          </div>
          {description ? (
            <Dialog.Description className="px-6 text-sm text-kumo-subtle max-sm:px-4">
              {description}
            </Dialog.Description>
          ) : null}
          {top ? <div className="px-6 pt-4 max-sm:px-4">{top}</div> : null}
          <div className="flex flex-col gap-4 overflow-y-auto overscroll-contain px-6 pt-4 pb-6 max-sm:px-4">
            {children}
          </div>
        </div>
      </Dialog>
    </Dialog.Root>
  );
}
