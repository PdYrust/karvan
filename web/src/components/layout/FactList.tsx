import type { ReactNode } from 'react';

export interface Fact {
  label: string;
  value: ReactNode;
}

/** Label and value rows split by hairlines, the dashboard's way of listing facts. */
export function FactList({ facts }: { facts: readonly Fact[] }) {
  return (
    <dl className="flex flex-col">
      {facts.map((f) => (
        <div
          key={f.label}
          className="flex items-baseline justify-between gap-4 border-b border-kumo-hairline py-2.5 last:border-0"
        >
          <dt className="shrink-0 text-sm text-kumo-subtle">{f.label}</dt>
          <dd className="min-w-0 text-end text-kumo-default tabular">{f.value}</dd>
        </div>
      ))}
    </dl>
  );
}
