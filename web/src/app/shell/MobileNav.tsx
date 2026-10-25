import { XIcon } from '@phosphor-icons/react';
import { useEffect, useRef, type ReactNode } from 'react';
import { Wordmark } from '../../components/brand/Wordmark';
import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';

/**
 * Sidebar as a sheet on small screens. A native <dialog> gives focus
 * trapping, Escape, backdrop dismiss and an inert page behind it. The sheet's
 * content mounts only while it is open, so a phone does not build the
 * navigation twice on every page load.
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
      className="m-0 h-dvh max-h-none w-[min(20rem,86vw)] max-w-none bg-kumo-canvas p-0 text-kumo-default shadow-xl backdrop:bg-kumo-contrast/40 me-auto lg:hidden"
    >
      <div className="flex h-full flex-col overflow-y-auto">
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-kumo-hairline ps-5 pe-2">
          <Wordmark />
          <button
            type="button"
            aria-label={t('closeMenu')}
            className="grid size-11 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint"
            onClick={onClose}
          >
            <XIcon size={20} aria-hidden="true" />
          </button>
        </div>
        {open ? children : null}
      </div>
    </dialog>
  );
}
