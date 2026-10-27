import { cn } from '@cloudflare/kumo/utils';
import { MagnifyingGlassIcon } from '@phosphor-icons/react';
/**
 * The search input at the start of a list toolbar (ListToolbar), drawn bare:
 * the toolbar's group draws its frame. `/` focuses it from anywhere on the
 * page (data-page-search), and Escape clears it.
 */
export function SearchField({
  label,
  value,
  onChange,
  className,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  className?: string;
}) {
  return (
    <div className={cn('relative min-w-0 flex-1', className)}>
      <MagnifyingGlassIcon
        size={16}
        aria-hidden="true"
        className="pointer-events-none absolute start-3 top-1/2 -translate-y-1/2 text-kumo-subtle"
      />
      <input
        type="search"
        aria-label={label}
        placeholder={label}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Escape' && value) {
            e.preventDefault();
            onChange('');
          }
        }}
        data-page-search=""
        autoComplete="off"
        spellCheck={false}
        className="h-full w-full rounded-lg bg-transparent ps-9 pe-3 text-base text-kumo-default placeholder:text-kumo-subtle focus-visible:-outline-offset-2"
      />
    </div>
  );
}
