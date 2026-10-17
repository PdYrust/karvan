import { cn } from '@cloudflare/kumo/utils';

/**
 * A thin meter that fills from the start edge. Decoration next to a number
 * in text, so it is hidden from screen readers.
 */
export function Bar({ share, className }: { share: number; className?: string }) {
  const width = Math.min(1, Math.max(0, share)) * 100;
  return (
    <span
      aria-hidden="true"
      className={cn('block h-1.5 overflow-hidden rounded-full bg-kumo-fill', className)}
    >
      <span
        className="block h-full rounded-full bg-chart-1"
        style={{ width: `${String(width)}%` }}
      />
    </span>
  );
}
