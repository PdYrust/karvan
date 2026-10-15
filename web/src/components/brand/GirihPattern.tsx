import { useId } from 'react';

// One khatam per tile. Neighbouring stars touch at their tips, which leaves
// the cross shapes between them: the classic khatam-o-salib tiling.
const tile = 64;
const r = tile / 2;
const star = Array.from({ length: 16 }, (_, i) => {
  const angle = (Math.PI / 8) * i - Math.PI / 2;
  const radius = i % 2 === 0 ? r : r * 0.7654;
  return `${(r + radius * Math.cos(angle)).toFixed(2)},${(r + radius * Math.sin(angle)).toFixed(2)}`;
}).join(' ');

/** Fine tile lattice for large quiet surfaces. Colored by currentColor. */
export function GirihPattern({ className }: { className?: string }) {
  const id = `girih-${useId().replace(/[^a-zA-Z0-9]/g, '')}`;
  return (
    <svg aria-hidden="true" className={className} width="100%" height="100%">
      <defs>
        <pattern id={id} width={tile} height={tile} patternUnits="userSpaceOnUse">
          <polygon points={star} fill="none" stroke="currentColor" strokeWidth="1" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill={`url(#${id})`} />
    </svg>
  );
}
