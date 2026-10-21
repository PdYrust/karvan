import type { ReactNode } from 'react';

export interface Fact {
  label: string;
  value: ReactNode;
}

/**
 * Label and value rows split by hairlines, the dashboard's way of listing
 * facts. A value too long for the rest of the row (a path on a phone)
 * moves under its label, still at the end, and may take the whole width.
 */
export function FactList({ facts }: { facts: readonly Fact[] }) {
  return (
    <dl className="flex flex-col">
      {facts.map((f) => (
        <div
          key={f.label}
          className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 border-b border-kumo-hairline py-2.5 last:border-0"
        >
          <dt className="shrink-0 text-sm text-kumo-subtle">{f.label}</dt>
          <dd className="ms-auto min-w-0 text-end text-kumo-default tabular">{f.value}</dd>
        </div>
      ))}
    </dl>
  );
}
