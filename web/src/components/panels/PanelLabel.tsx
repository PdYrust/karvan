import { cn } from '@cloudflare/kumo/utils';
import { ArrowRightIcon } from '@phosphor-icons/react';
import type { PanelKind } from '../../lib/api/panels';
import { PanelLogo, panelNames } from './PanelLogo';

/** A panel's logo, on a white tile for PasarGuard so it reads in both themes. */
export function PanelMark({ kind, size = 16 }: { kind: PanelKind; size?: number }) {
  if (kind === 'pasarguard') {
    return (
      <span className="inline-grid shrink-0 place-items-center">
        <PanelLogo kind="pasarguard" size={size} />
      </span>
    );
  }
  return (
    <span className="inline-grid shrink-0 place-items-center text-kumo-default">
      <PanelLogo kind="marzban" size={size} />
    </span>
  );
}

/** "logo PasarGuard 5.4.1", or a folder name with the kind's logo. */
export function PanelLabel({
  kind,
  name,
  version,
  size = 16,
  className,
}: {
  kind: PanelKind | undefined;
  /** Shown instead of the kind's name, such as the panel's folder. */
  name?: string;
  version?: string;
  size?: number;
  className?: string;
}) {
  return (
    <span className={cn('inline-flex min-w-0 items-center gap-2', className)}>
      {kind ? <PanelMark kind={kind} size={size} /> : null}
      <span className="truncate">{name ?? (kind ? panelNames[kind] : '')}</span>
      {version ? (
        <bdi dir="ltr" className="shrink-0 text-sm text-kumo-subtle tabular">
          {version}
        </bdi>
      ) : null}
    </span>
  );
}

/** A move between panels: source logo, an arrow in the reading direction, destination logo. */
export function MoveLabel({
  from,
  to,
  label,
  size = 16,
}: {
  from: PanelKind;
  to: PanelKind;
  /** Read out instead of the two names, such as "Marzban to PasarGuard". */
  label: string;
  size?: number;
}) {
  return (
    <span role="img" aria-label={label} className="inline-flex items-center gap-1.5">
      <PanelMark kind={from} size={size} />
      <ArrowRightIcon size={12} aria-hidden="true" className="text-kumo-subtle rtl:-scale-x-100" />
      <PanelMark kind={to} size={size} />
    </span>
  );
}
