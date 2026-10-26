import { cn } from '@cloudflare/kumo/utils';
import type { ReactNode } from 'react';

/**
 * One end of the scene: a panel or the backup, its name, a second line and
 * a number. Standing (`row`, on phones) it is a wide tile: the mark, the
 * name and the line under it at the start, the number at the end.
 */
export function SceneTile({
  mark,
  name,
  detail,
  value,
  dim = false,
  row = false,
}: {
  mark: ReactNode;
  name: ReactNode;
  /** The address or folder under the name, a machine value. */
  detail?: ReactNode;
  value?: ReactNode;
  /** An offline panel. */
  dim?: boolean;
  row?: boolean;
}) {
  return (
    <div
      className={cn(
        'relative z-10 flex min-w-0 gap-1 rounded-lg bg-kumo-base ring ring-kumo-hairline',
        row
          ? 'flex-row items-center gap-3 px-3 py-2.5 text-start'
          : 'flex-col items-center px-2 py-3 text-center',
        dim && 'bg-kumo-recessed',
      )}
    >
      <span className={cn('flex h-8 shrink-0 items-center', dim && 'opacity-50')}>{mark}</span>
      <span
        className={cn(
          'flex min-w-0 max-w-full flex-col',
          row ? 'flex-1 items-start' : 'items-center gap-1',
        )}
      >
        <span className="max-w-full truncate text-sm font-medium text-kumo-strong">{name}</span>
        {detail ? (
          <bdi dir="ltr" className="max-w-full truncate font-mono text-tiny text-kumo-subtle">
            {detail}
          </bdi>
        ) : null}
      </span>
      {value ? (
        <span className={cn('text-sm text-kumo-default tabular', row && 'shrink-0 text-end')}>
          {value}
        </span>
      ) : null}
    </div>
  );
}
