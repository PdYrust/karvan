import { dashRings, starRings, viewBox, type Ring } from './logo-geometry';

function pathOf(rings: readonly Ring[]): string {
  return rings.map((r) => 'M' + r.map(([x, y]) => `${x} ${y}`).join('L') + 'Z').join(' ');
}

const starPath = pathOf(starRings);
const dashPath = pathOf(dashRings);

interface MarkProps {
  size?: number | string;
  className?: string;
  /** Decorative by default; give a title when the mark stands alone. */
  title?: string;
}

/** The Karvan star. Never mirrored in RTL. */
export function Mark({ size = 32, className, title }: MarkProps) {
  return (
    <svg
      viewBox={`0 0 ${viewBox} ${viewBox}`}
      width={size}
      height={size}
      className={className}
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : true}
      aria-label={title}
    >
      <path className="fill-karvan-firouzeh" d={starPath} />
      <path className="fill-karvan-saffron" d={dashPath} />
    </svg>
  );
}
