import { KumoPortalProvider } from '@cloudflare/kumo/utils';
import { XIcon } from '@phosphor-icons/react';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Mark } from '../../components/brand/Mark';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { shellMessages } from './messages';
import { PanelSwitcher } from './PanelSwitcher';

/**
 * Sidebar as a sheet on small screens. A native <dialog> gives focus
 * trapping, Escape, backdrop dismiss and an inert page behind it. The sheet's
 * content mounts when it first opens, so a phone does not build the
 * navigation twice on every page load, and then stays, so the sheet still
 * has its rows while it slides out. A modal dialog sits in the top layer
 * and makes the rest of the page inert, so menus opened inside it (the panel
 * switcher) portal into the dialog, or they would sit under it and ignore taps.
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
  const ts = useT(sharedMessages);
  const ref = useRef<HTMLDialogElement>(null);
  const [opened, setOpened] = useState(open);
  if (open && !opened) setOpened(true);

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
      className="k-sheet m-0 h-dvh max-h-none w-[min(20rem,86vw)] max-w-none bg-kumo-canvas p-0 text-kumo-default shadow-xl backdrop:bg-kumo-contrast/40 me-auto lg:hidden"
    >
      <KumoPortalProvider container={ref}>
        <div className="flex h-full flex-col overflow-y-auto">
          <div className="flex h-14 shrink-0 items-center gap-2 border-b border-kumo-hairline ps-3 pe-2">
            <Mark size={28} title={ts('brand')} />
            {opened ? <PanelSwitcher /> : <span className="flex-1" />}
            <button
              type="button"
              aria-label={t('closeMenu')}
              className="grid size-11 shrink-0 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint"
              onClick={onClose}
            >
              <XIcon size={20} aria-hidden="true" />
            </button>
          </div>
          {opened ? children : null}
        </div>
      </KumoPortalProvider>
    </dialog>
  );
}
