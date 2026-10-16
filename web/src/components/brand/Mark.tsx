import { cn } from '@cloudflare/kumo/utils';

interface MarkProps {
  size?: number | string;
  className?: string;
  /** Decorative by default; give a title when the mark stands alone. */
  title?: string;
}

/**
 * The Karvan mark: two panels overlapping into one. Same drawing as
 * web/public/favicon.svg and docs/brand/karvan-mark.svg. Never mirrored in RTL.
 */
export function Mark({ size = 28, className, title }: MarkProps) {
  return (
    <svg
      viewBox="0 0 64 64"
      width={size}
      height={size}
      className={cn('shrink-0', className)}
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : true}
      aria-label={title}
    >
      <rect width="64" height="64" rx="14" className="fill-kumo-brand" />
      <g className="stroke-on-brand" strokeWidth="3.2" fill="none">
        <rect x="12" y="22" width="24" height="24" rx="5" />
        <rect x="28" y="18" width="24" height="24" rx="5" />
      </g>
      <rect x="28" y="22" width="8" height="20" className="fill-on-brand" />
    </svg>
  );
}
