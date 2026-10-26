import { cn } from '@cloudflare/kumo/utils';
import { ArrowRightIcon, MagnifyingGlassIcon } from '@phosphor-icons/react';
import { useId, useState, type KeyboardEvent } from 'react';
import { useCommands, type Command } from '../../../app/shell/commands';
import { shellMessages } from '../../../app/shell/messages';
import { useShell } from '../../../app/shell/outlet';
import { useT } from '../../../i18n/locale';
import { usePanels } from '../../panels/use-panels';
import { moveActive, searchGroups, searchRows } from './search-rows';
import { useRecentCommands } from './use-recent-commands';
const kbd = 'rounded border border-kumo-hairline bg-kumo-base px-1.5 py-0.5 font-mono text-tiny';
function Row({
  id,
  command,
  active,
  onPoint,
  onRun,
}: {
  id: string;
  command: Command;
  active: boolean;
  onPoint: () => void;
  onRun: () => void;
}) {
  return (
    // The field keeps focus (the row never takes it); the keys there or a press choose a row.
    <div
      id={id}
      role="option"
      tabIndex={-1}
      aria-selected={active}
      onPointerMove={onPoint}
      onPointerDown={(e) => e.preventDefault()}
      onClick={onRun}
      onKeyDown={(e) => {
        if (e.key === 'Enter') onRun();
      }}
      className={cn(
        'flex h-9 cursor-pointer items-center gap-3 rounded-md px-3 pointer-coarse:h-11',
        active && 'bg-kumo-tint',
      )}
    >
      <span className="grid size-[18px] shrink-0 place-items-center text-kumo-subtle">
        {command.icon}
      </span>
      <span className="flex min-w-0 flex-1 items-baseline gap-1.5">
        <span className="truncate text-kumo-strong">{command.title}</span>
        {command.hint ? (
          <span className="truncate text-sm text-kumo-subtle">- {command.hint}</span>
        ) : null}
      </span>
      {active ? (
        <ArrowRightIcon
          size={16}
          aria-hidden="true"
          className="shrink-0 text-kumo-subtle rtl:-scale-x-100"
        />
      ) : null}
    </div>
  );
}
/**
 * The overview's search, like the dashboard home's: a wide field in a soft
 * halo, and a panel under it with the recent pages, then what the palette
 * finds as you type. The field keeps focus; the keys walk the rows.
 */
export function HomeSearch() {
  const t = useT(shellMessages);
  const { openShortcuts } = useShell();
  const id = useId();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [active, setActive] = useState(0);
  const panels = usePanels().data ?? [];
  const recents = useRecentCommands(panels);
  const groups = searchGroups(recents, useCommands(openShortcuts, open), query);
  const rows = searchRows(groups);
  const current = Math.min(active, rows.length - 1);
  const rowId = (i: number) => `${id}-row-${String(i)}`;
  function close() {
    setOpen(false);
    setActive(0);
  }
  function run(c: Command) {
    close();
    setQuery('');
    c.run();
  }
  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') {
      if (open) e.preventDefault();
      close();
      return;
    }
    if (e.key === 'Enter') {
      const c = rows[current];
      if (open && c) {
        e.preventDefault();
        run(c);
      }
      return;
    }
    // Home and End move the caret in the field; only the arrows walk the rows.
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
    e.preventDefault();
    if (!open) {
      setOpen(true);
      return;
    }
    const next = moveActive(current, e.key, rows.length);
    if (next !== undefined) {
      setActive(next);
      document.getElementById(rowId(next))?.scrollIntoView({ block: 'nearest' });
    }
  }
  const shown = open && (rows.length > 0 || query.trim() !== '');
  let index = 0;
  return (
    <div className="relative w-full">
      {/* The halo: a second, tinted ring a few px outside the field. */}
      <div className="rounded-xl bg-kumo-tint p-1 ring ring-kumo-line">
        <div className="relative">
          <MagnifyingGlassIcon
            size={18}
            aria-hidden="true"
            className="pointer-events-none absolute start-3 top-1/2 -translate-y-1/2 text-kumo-subtle"
          />
          <input
            type="text"
            role="combobox"
            aria-label={t('search')}
            aria-expanded={shown}
            aria-controls={`${id}-list`}
            aria-autocomplete="list"
            aria-activedescendant={shown && rows.length > 0 ? rowId(current) : undefined}
            placeholder={t('search')}
            autoComplete="off"
            enterKeyHint="search"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setActive(0);
              setOpen(true);
            }}
            onFocus={() => setOpen(true)}
            onBlur={close}
            onClick={() => setOpen(true)}
            onKeyDown={onKeyDown}
            className="h-11 w-full rounded-lg bg-kumo-base ps-10 pe-3 text-kumo-default ring ring-kumo-line outline-none placeholder:text-kumo-subtle focus-visible:ring-2 focus-visible:ring-kumo-focus pointer-coarse:h-11"
          />
        </div>
      </div>
      <div
        data-closed={shown ? undefined : ''}
        className="k-drop absolute inset-x-0 top-full z-20 mt-2 overflow-hidden rounded-lg bg-kumo-base shadow-lg ring ring-kumo-hairline"
      >
        <div
          id={`${id}-list`}
          role="listbox"
          aria-label={t('search')}
          className="max-h-[min(24rem,60dvh)] overflow-y-auto overscroll-contain p-1.5"
        >
          {rows.length === 0 ? (
            <p className="px-3 py-2 text-sm text-kumo-subtle">{t('searchEmpty')}</p>
          ) : null}
          {groups.map((g) => (
            <div key={g.id} role="group" aria-labelledby={`${id}-${g.id}`} className="pb-1">
              <div id={`${id}-${g.id}`} className="px-3 pt-2 pb-1 text-sm text-kumo-subtle">
                {g.label}
              </div>
              {g.items.map((c) => {
                const i = index++;
                return (
                  <Row
                    key={c.id}
                    id={rowId(i)}
                    command={c}
                    active={i === current}
                    onPoint={() => setActive(i)}
                    onRun={() => run(c)}
                  />
                );
              })}
            </div>
          ))}
        </div>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-kumo-hairline bg-kumo-tint px-3 py-2 text-sm text-kumo-subtle">
          <span className="flex items-center gap-1.5">
            <kbd className={kbd}>↑ ↓</kbd>
            {t('paletteNavigate')}
          </span>
          <span className="flex items-center gap-1.5">
            <kbd className={kbd}>↵</kbd>
            {t('paletteSelect')}
          </span>
          <span className="flex items-center gap-1.5">
            <kbd className={kbd}>Esc</kbd>
            {t('paletteClose')}
          </span>
        </div>
      </div>
    </div>
  );
}
