import { cn } from '@cloudflare/kumo/utils';

/**
 * A thin meter that fills from the start edge. Decoration next to a number
 * in text, so it is hidden from screen readers.
 */
export function Bar({
  share,
  grow = false,
  className,
}: {
  share: number;
  /** Fill in from the start edge when it first shows: data, not progress (DESIGN.md section 5). */
  grow?: boolean;
  className?: string;
}) {
  const width = Math.min(1, Math.max(0, share)) * 100;
  return (
    <span
      aria-hidden="true"
      className={cn('block h-1.5 overflow-hidden rounded-full bg-kumo-fill', className)}
    >
      <span
        className={cn('block h-full rounded-full bg-chart-1', grow && 'k-fill')}
        style={{ width: `${String(width)}%` }}
      />
    </span>
  );
}
