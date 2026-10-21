import { Input } from '@cloudflare/kumo/components/input';
import { cn } from '@cloudflare/kumo/utils';
import { MagnifyingGlassIcon } from '@phosphor-icons/react';

/**
 * The search field across the top of a list. `/` focuses it from anywhere
 * on the page (data-page-search), and Escape clears it.
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
    <div className={cn('relative', className)}>
      <MagnifyingGlassIcon
        size={16}
        aria-hidden="true"
        className="pointer-events-none absolute start-3 top-1/2 -translate-y-1/2 text-kumo-subtle"
      />
      <Input
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
        className="w-full ps-9 pointer-coarse:h-11"
      />
    </div>
  );
}
