import { useRef, useState } from 'react';

/** How long after a press an opening still counts as its answer, in ms. */
const pressWindow = 500;

interface Change {
  reason?: string;
}

/**
 * A submenu that a second press on its trigger closes again, as the root menu's trigger does.
 * A submenu opened by hovering stays open on the first press, so a mouse click never closes
 * what the pointer just opened; the next press closes it.
 * Spread `root` on `DropdownMenu.Sub` and `trigger` on its `SubTrigger`.
 */
export function useSubToggle() {
  const [open, setOpenState] = useState(false);
  // Refs, because the menu calls back from timers (hover) after the render that made them.
  const isOpen = useRef(false);
  // Whether a press has opened or kept the submenu since it last opened.
  const pressed = useRef(false);
  // The state when the pointer went down, before the menu reacts to that press.
  const atDown = useRef<boolean | null>(null);
  // Set while a press is closing the submenu, so the same press cannot open it again.
  const closing = useRef(false);
  // When the trigger was last pressed while closed: a mouse press often lets the hover timer
  // open the submenu a moment later, and that opening still answers the press.
  const pressedAt = useRef(-Infinity);

  const setOpen = (next: boolean) => {
    isOpen.current = next;
    if (!next) pressed.current = false;
    setOpenState(next);
  };

  const onOpenChange = (next: boolean, details?: Change) => {
    if (!next) return setOpen(false);
    if (closing.current) return;
    if (isOpen.current) {
      if (details?.reason === 'trigger-press') pressed.current = true;
      return;
    }
    const answersPress = performance.now() - pressedAt.current < pressWindow;
    setOpen(true);
    pressed.current = details?.reason === 'trigger-press' || answersPress;
  };

  const onPointerDownCapture = () => {
    atDown.current = isOpen.current && pressed.current;
  };

  const onClickCapture = () => {
    const wasPressedOpen = atDown.current ?? (isOpen.current && pressed.current);
    atDown.current = null;
    if (!wasPressedOpen) {
      if (isOpen.current) pressed.current = true;
      else pressedAt.current = performance.now();
      return;
    }
    closing.current = true;
    setOpen(false);
    // The menu handles this press after the capture phase; let it pass, then listen again.
    setTimeout(() => {
      closing.current = false;
    }, 0);
  };

  return {
    root: { open, onOpenChange },
    trigger: { onPointerDownCapture, onClickCapture },
  };
}
