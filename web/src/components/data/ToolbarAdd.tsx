import { Button } from '@cloudflare/kumo/components/button';
import { PlusIcon } from '@phosphor-icons/react';

/**
 * A list toolbar's add action, after the search: "+" and its label from 640 px,
 * "+" alone on phones, 36 px square, named by the same words.
 */
export function ToolbarAdd({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <Button
      variant="primary"
      aria-label={label}
      icon={<PlusIcon size={16} />}
      className="shrink-0 pointer-coarse:h-9 max-sm:w-9 max-sm:justify-center max-sm:px-0"
      onClick={onClick}
    >
      <span className="max-sm:sr-only">{label}</span>
    </Button>
  );
}
