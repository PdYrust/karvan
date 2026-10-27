import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { cn } from '@cloudflare/kumo/utils';
import { CaretUpDownIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import type { Choice } from '../../lib/lists/choices';
import { useFormat } from '../../lib/preferences/region';
import { segment } from './segment';
export interface ChoiceOption {
  value: string;
  label: ReactNode;
  /** The label as words, for the trigger's name. */
  text: string;
  /** How many rows the option shows, where Karvan knows it. */
  count?: number;
}
/** A filter or a sort of a list toolbar. */
export interface ToolbarChoice extends Choice {
  /** Names it: "State", "Sort by". */
  label: string;
  options: readonly ChoiceOption[];
  onChange: (value: string) => void;
}
export const currentOf = (c: ToolbarChoice) => c.options.find((o) => o.value === c.value);
/** An option's label, and its count quiet at the end. */
export function OptionText({ option }: { option: ChoiceOption }) {
  const fmt = useFormat();
  return (
    <>
      <span className="min-w-0 truncate">{option.label}</span>
      {option.count === undefined ? null : (
        <span className="ms-auto ps-4 text-kumo-subtle tabular">{fmt.number(option.count)}</span>
      )}
    </>
  );
}
/**
 * One segment of a list toolbar from 640 px: the current choice and an up and down chevron;
 * it opens a small menu with a check on the current choice and each choice's count.
 */
export function ChoiceMenu({ choice, className }: { choice: ToolbarChoice; className?: string }) {
  const current = currentOf(choice);
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <button
            type="button"
            aria-label={`${choice.label}: ${current?.text ?? ''}`}
            className={cn(segment, 'max-w-56 gap-2 px-3', className)}
          >
            <span className="flex min-w-0 items-center gap-2">
              <span className="min-w-0 truncate">{current?.label}</span>
            </span>
            <CaretUpDownIcon size={14} aria-hidden="true" className="shrink-0 text-kumo-subtle" />
          </button>
        }
      />
      <DropdownMenu.Content align="end" className="min-w-48">
        <DropdownMenu.Group>
          <DropdownMenu.Label>{choice.label}</DropdownMenu.Label>
          <DropdownMenu.RadioGroup
            value={choice.value}
            onValueChange={(v: string) => choice.onChange(v)}
          >
            {choice.options.map((o) => (
              <DropdownMenu.RadioItem key={o.value} value={o.value} closeOnClick>
                <span className="flex w-full items-center gap-2">
                  <OptionText option={o} />
                  <span
                    className={cn(
                      'flex w-4 shrink-0 justify-center',
                      o.count === undefined && 'ms-auto',
                    )}
                  >
                    <DropdownMenu.RadioItemIndicator />
                  </span>
                </span>
              </DropdownMenu.RadioItem>
            ))}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Group>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
