import { Tooltip } from '@cloudflare/kumo/components/tooltip';
import { cn } from '@cloudflare/kumo/utils';
import { CaretDownIcon, CaretRightIcon } from '@phosphor-icons/react';
import { useId, useState } from 'react';
import { NavLink, useLocation } from 'react-router';
import { Collapse } from '../../components/layout/Collapse';
import { usePanels } from '../../features/panels/use-panels';
import { useLocale, useT } from '../../i18n/locale';
import { shellMessages } from './messages';
import { nav, type NavItem } from './nav';
// As dense as the dashboard's rows: 30 high, 32 apart. On touch screens a row draws 40 high,
// 44 apart, and its hit area reaches 2 px past each edge, so every row takes a 44 px tap.
// Hover colors change at once (Kumo's rule), so no transition here.
const row =
  "relative flex min-h-7.5 w-full items-center gap-3 rounded-md px-3 text-start text-base text-kumo-default hover:bg-kumo-tint aria-[current=page]:bg-kumo-tint aria-[current=page]:font-medium aria-[current=page]:text-kumo-strong pointer-coarse:min-h-10 pointer-coarse:before:absolute pointer-coarse:before:inset-x-0 pointer-coarse:before:-inset-y-0.5 pointer-coarse:before:content-['']";
const rows = 'flex flex-col gap-0.5 pointer-coarse:gap-1';
const iconRow =
  'grid size-10 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint aria-[current=page]:bg-kumo-tint pointer-coarse:size-11';
/** Analytics' sub-pages, as its tabs: this server, then each panel. */
function SubRows({ onNavigate }: { onNavigate?: () => void }) {
  const t = useT(shellMessages);
  const panels = (usePanels().data ?? []).filter((p) => p.id > 0);
  return (
    // The guide line sits under the parent's icon; sub-labels line up with its label.
    <ul className={cn('ms-5 mt-0.5 border-s border-kumo-hairline ps-2', rows)}>
      <li>
        <NavLink to="/analytics/server" onClick={onNavigate} className={row}>
          {t('navServer')}
        </NavLink>
      </li>
      {panels.map((p) => (
        <li key={p.id}>
          <NavLink to={`/analytics/panels/${String(p.id)}`} onClick={onNavigate} className={row}>
            <bdi dir="ltr" className="truncate">
              {p.name}
            </bdi>
          </NavLink>
        </li>
      ))}
    </ul>
  );
}
function NavRow({
  item,
  collapsed,
  onNavigate,
}: {
  item: NavItem;
  collapsed: boolean;
  onNavigate?: () => void;
}) {
  const t = useT(shellMessages);
  const { dir } = useLocale();
  const Icon = item.icon;
  const label = t(item.label);
  if (collapsed) {
    return (
      <Tooltip content={label} side={dir === 'rtl' ? 'left' : 'right'} asChild>
        <NavLink
          to={item.to}
          end={item.to === '/'}
          aria-label={label}
          onClick={onNavigate}
          className={iconRow}
        >
          <Icon size={18} aria-hidden="true" />
        </NavLink>
      </Tooltip>
    );
  }
  if (item.perPanel) return <GroupRow item={item} label={label} onNavigate={onNavigate} />;
  return (
    <NavLink to={item.to} end={item.to === '/'} onClick={onNavigate} className={row}>
      <Icon size={16} aria-hidden="true" className="shrink-0 text-kumo-subtle" />
      <span>{label}</span>
    </NavLink>
  );
}

/**
 * A row with sub-pages is a toggle: it opens and closes its sub-rows in place and goes nowhere.
 * It starts open while one of its pages is shown; the reader can close it. The choice holds
 * until the reader moves into or out of the section.
 */
function GroupRow({
  item,
  label,
  onNavigate,
}: {
  item: NavItem;
  label: string;
  onNavigate?: () => void;
}) {
  const { pathname } = useLocation();
  const id = useId();
  const inSection = pathname.startsWith(item.to);
  const [chosen, setChosen] = useState<{ inSection: boolean; open: boolean } | null>(null);
  const open = chosen?.inSection === inSection ? chosen.open : inSection;
  const Icon = item.icon;
  const Caret = open ? CaretDownIcon : CaretRightIcon;
  return (
    <>
      <button
        type="button"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setChosen({ inSection, open: !open })}
        className={row}
      >
        <Icon size={16} aria-hidden="true" className="shrink-0 text-kumo-subtle" />
        <span>{label}</span>
        <Caret size={14} aria-hidden="true" className="ms-auto text-kumo-subtle rtl:-scale-x-100" />
      </button>
      <Collapse id={id} open={open}>
        <SubRows onNavigate={onNavigate} />
      </Collapse>
    </>
  );
}

export function SidebarNav({
  collapsed = false,
  onNavigate,
}: {
  collapsed?: boolean;
  onNavigate?: () => void;
}) {
  const t = useT(shellMessages);
  return (
    <nav
      aria-label={t('mainNav')}
      className={cn('flex flex-col gap-3', collapsed && 'items-center gap-2')}
    >
      {nav.map((group, i) => (
        <div key={group.label ?? i} className={rows}>
          {group.label && !collapsed ? (
            <h2 className="px-3 pt-1 pb-0.5 text-sm text-kumo-subtle">{t(group.label)}</h2>
          ) : null}
          {group.label && collapsed ? (
            <hr aria-hidden="true" className="mx-2 my-1 border-kumo-hairline" />
          ) : null}
          <ul className={rows}>
            {group.items.map((item) => (
              <li key={item.label}>
                <NavRow item={item} collapsed={collapsed} onNavigate={onNavigate} />
              </li>
            ))}
          </ul>
        </div>
      ))}
    </nav>
  );
}
