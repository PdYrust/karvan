import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { CaretDownIcon } from '@phosphor-icons/react';
import { useNavigate } from 'react-router';
import { StatusBadge, StatusLabel, type StatusTone } from '../../components/feedback/StatusBadge';

export interface Problem {
  key: string;
  tone: StatusTone;
  text: string;
}

/**
 * A row's state: an icon and a word in a table, a badge on a phone card. With
 * problems it opens a small menu like the account menu, one item per problem,
 * worst first; a press opens `to`, where it is fixed.
 */
export function StateMenu({
  word,
  tone,
  problems,
  label,
  to,
  badge = false,
}: {
  word: string;
  tone: StatusTone;
  problems: readonly Problem[];
  /** Names the button: the row and its state. */
  label: string;
  to?: string;
  badge?: boolean;
}) {
  const navigate = useNavigate();
  const look = badge ? (
    <StatusBadge tone={tone}>{word}</StatusBadge>
  ) : (
    <StatusLabel tone={tone}>{word}</StatusLabel>
  );
  if (problems.length === 0) return look;
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <button
            type="button"
            aria-label={label}
            className="relative inline-flex cursor-pointer items-center gap-1 rounded-md after:absolute after:-inset-x-1 after:-inset-y-3 after:content-[''] hover:opacity-80 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-kumo-brand"
          >
            {look}
            <CaretDownIcon size={12} aria-hidden="true" className="text-kumo-subtle" />
          </button>
        }
      />
      <DropdownMenu.Content align="start" className="w-72 max-w-[calc(100vw-2rem)]">
        {problems.map((p) => (
          <DropdownMenu.Item key={p.key} onClick={to ? () => void navigate(to) : undefined}>
            <span className="text-sm whitespace-normal">
              <StatusLabel tone={p.tone}>{p.text}</StatusLabel>
            </span>
          </DropdownMenu.Item>
        ))}
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
