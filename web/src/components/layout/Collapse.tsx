import { useState, type ReactNode } from 'react';

/**
 * Content that opens and closes in place, moving both ways (DESIGN.md section 5).
 * It mounts on the first open and then stays, so it can close with its motion;
 * closed, it is inert: out of the tab order and hidden from screen readers.
 */
export function Collapse({
  id,
  open,
  children,
}: {
  id: string;
  open: boolean;
  children: ReactNode;
}) {
  const [used, setUsed] = useState(open);
  if (open && !used) setUsed(true);
  return (
    <div id={id} className="k-collapse" data-closed={open ? undefined : ''} inert={!open}>
      <div>{used ? children : null}</div>
    </div>
  );
}
