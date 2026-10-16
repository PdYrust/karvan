import { Button } from '@cloudflare/kumo/components/button';
import { XIcon } from '@phosphor-icons/react';
import { useEffect, useRef, type ReactNode } from 'react';
import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';

/**
 * Sidebar as a sheet on small screens. A native <dialog> gives focus
 * trapping, Escape, backdrop dismiss and an inert page behind it.
 */
export function MobileNav({
  open,
  onClose,
  children,
}: {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}) {
  const t = useT(shellMessages);
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) d.showModal();
    if (!open && d.open) d.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      onClose={onClose}
      // Light dismiss: a click on the backdrop closes the sheet, like Escape.
      {...{ closedby: 'any' }}
      aria-label={t('mainNav')}
      className="m-0 h-dvh max-h-none w-[min(20rem,86vw)] max-w-none bg-kumo-base p-0 text-kumo-default shadow-xl backdrop:bg-kumo-contrast/40 me-auto lg:hidden"
    >
      <div className="flex h-full flex-col gap-4 overflow-y-auto p-4">
        <div className="flex justify-end">
          <Button
            variant="ghost"
            shape="square"
            aria-label={t('closeMenu')}
            className="size-11"
            icon={<XIcon size={20} />}
            onClick={onClose}
          />
        </div>
        {children}
      </div>
    </dialog>
  );
}
