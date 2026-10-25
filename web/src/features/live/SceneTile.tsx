import { cn } from '@cloudflare/kumo/utils';
import type { ReactNode } from 'react';

/** One end of the scene: a panel or the backup, its name, a second line and a number. */
export function SceneTile({
  mark,
  name,
  detail,
  value,
  dim = false,
}: {
  mark: ReactNode;
  name: ReactNode;
  /** The address or folder under the name, a machine value. */
  detail?: ReactNode;
  value?: ReactNode;
  /** An offline panel. */
  dim?: boolean;
}) {
  return (
    <div
      className={cn(
        'relative z-10 flex min-w-0 flex-col items-center gap-1 rounded-lg bg-kumo-base px-2 py-3 text-center ring ring-kumo-hairline',
        dim && 'bg-kumo-recessed',
      )}
    >
      <span className={cn('flex h-8 items-center', dim && 'opacity-50')}>{mark}</span>
      <span className="max-w-full truncate text-sm font-medium text-kumo-strong">{name}</span>
      {detail ? (
        <bdi dir="ltr" className="max-w-full truncate font-mono text-tiny text-kumo-subtle">
          {detail}
        </bdi>
      ) : null}
      {value ? <span className="text-sm text-kumo-default tabular">{value}</span> : null}
    </div>
  );
}
