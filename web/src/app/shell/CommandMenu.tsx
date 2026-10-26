import { CommandPalette } from '@cloudflare/kumo/components/command-palette';
import { cn } from '@cloudflare/kumo/utils';
import { useState, type RefObject } from 'react';
import { useT } from '../../i18n/locale';
import { filterCommands, useCommands, type Command, type CommandGroup } from './commands';
import { shellMessages } from './messages';

const kbd = 'rounded border border-kumo-hairline bg-kumo-base px-1.5 py-0.5 font-mono text-tiny';

export function CommandMenu({
  open,
  onOpenChange,
  onShortcuts,
  opener,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onShortcuts: () => void;
  /** What had focus when the palette opened; closing it without a choice goes back there. */
  opener: RefObject<HTMLElement | null>;
}) {
  const t = useT(shellMessages);
  const [query, setQuery] = useState('');
  const groups = filterCommands(useCommands(onShortcuts, open), query);

  // A chosen command decides where focus goes (a new page, a dialog), so only a dismissal returns it.
  function run(c: Command) {
    onOpenChange(false);
    setQuery('');
    c.run();
  }

  return (
    <CommandPalette.Root
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (o) return;
        setQuery('');
        const back = opener.current;
        // After the palette has let go of focus, which would leave it on the page body.
        requestAnimationFrame(() => {
          if (back?.isConnected) back.focus();
        });
      }}
      items={groups}
      value={query}
      onValueChange={setQuery}
      itemToStringValue={(g: CommandGroup) => g.label}
      getSelectableItems={(gs: CommandGroup[]) => gs.flatMap((g) => g.items)}
      onSelect={(c: Command) => run(c)}
    >
      <CommandPalette.Input
        placeholder={t('searchPlaceholder')}
        aria-label={t('search')}
        trailing={
          <kbd title={t('paletteClose')} className={cn(kbd, 'shrink-0 text-kumo-subtle')}>
            Esc
          </kbd>
        }
      />
      <CommandPalette.List>
        <CommandPalette.Results>
          {(g: CommandGroup) => (
            <CommandPalette.Group key={g.id} items={g.items}>
              <CommandPalette.GroupLabel>{g.label}</CommandPalette.GroupLabel>
              <CommandPalette.Items>
                {(c: Command) => (
                  <CommandPalette.Item key={c.id} value={c} onClick={() => run(c)}>
                    <span className="flex items-center gap-3">
                      <span className="grid size-[18px] place-items-center text-kumo-subtle">
                        {c.icon}
                      </span>
                      <span>{c.title}</span>
                      {c.hint ? <span className="text-sm text-kumo-subtle">{c.hint}</span> : null}
                    </span>
                  </CommandPalette.Item>
                )}
              </CommandPalette.Items>
            </CommandPalette.Group>
          )}
        </CommandPalette.Results>
        <CommandPalette.Empty>{t('searchEmpty')}</CommandPalette.Empty>
      </CommandPalette.List>
      <CommandPalette.Footer>
        <span className="flex items-center gap-2">
          <kbd className={kbd}>↑↓</kbd>
          <span>{t('paletteNavigate')}</span>
        </span>
        <span className="flex items-center gap-2">
          <kbd className={kbd}>Enter</kbd>
          <span>{t('paletteSelect')}</span>
        </span>
      </CommandPalette.Footer>
    </CommandPalette.Root>
  );
}
