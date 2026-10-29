import { Badge } from '@cloudflare/kumo/components/badge';
import { cn } from '@cloudflare/kumo/utils';
import { CaretRightIcon } from '@phosphor-icons/react';

export interface ChoiceRow<V extends string> {
  value: V;
  title: string;
  hint: string;
  /** Marks the recommended answer. */
  badge?: string;
  disabled?: boolean;
}

/**
 * One question's answers as rows in a ringed box split by hairlines, each a
 * button with a chevron: a press answers and goes on, so the step has no
 * Continue. The answer given before (after Back) stays tinted.
 */
export function ChoiceList<V extends string>({
  label,
  rows,
  chosen,
  onChoose,
}: {
  label: string;
  rows: readonly ChoiceRow<V>[];
  chosen?: V;
  onChoose: (v: V) => void;
}) {
  return (
    <ul
      aria-label={label}
      className="flex flex-col divide-y divide-kumo-hairline overflow-hidden rounded-lg bg-kumo-base ring ring-kumo-hairline"
    >
      {rows.map((r) => (
        <li key={r.value}>
          <button
            type="button"
            disabled={r.disabled}
            aria-current={r.value === chosen ? 'true' : undefined}
            onClick={() => onChoose(r.value)}
            className={cn(
              'flex min-h-16 w-full items-center gap-4 px-4 py-3 text-start focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-kumo-brand',
              r.disabled ? 'cursor-not-allowed opacity-50' : 'hover:bg-kumo-tint',
              r.value === chosen && 'bg-kumo-tint',
            )}
          >
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span className="font-medium text-kumo-strong">{r.title}</span>
                {r.badge ? <Badge variant="outline">{r.badge}</Badge> : null}
              </span>
              <span className="text-sm text-kumo-subtle">{r.hint}</span>
            </span>
            <CaretRightIcon
              size={16}
              aria-hidden="true"
              className="shrink-0 text-kumo-subtle rtl:-scale-x-100"
            />
          </button>
        </li>
      ))}
    </ul>
  );
}
