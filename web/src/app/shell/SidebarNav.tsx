import { Tooltip } from '@cloudflare/kumo/components/tooltip';
import { cn } from '@cloudflare/kumo/utils';
import { CaretDownIcon, CaretRightIcon } from '@phosphor-icons/react';
import { NavLink, useLocation } from 'react-router';
import { usePanels } from '../../features/panels/use-panels';
import { useLocale, useT } from '../../i18n/locale';
import { shellMessages } from './messages';
import { nav, type NavItem } from './nav';
// As dense as the dashboard's rows: 32 high, 34 apart, 44 on touch screens.
// Hover colors change at once (Kumo's rule), so no transition here.
const row =
  'flex min-h-8 items-center gap-3 rounded-md px-3 text-base pointer-coarse:min-h-11 text-kumo-default hover:bg-kumo-tint aria-[current=page]:bg-kumo-tint aria-[current=page]:font-medium aria-[current=page]:text-kumo-strong';
const iconRow =
  'grid size-10 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint aria-[current=page]:bg-kumo-tint pointer-coarse:size-11';
/** Analytics' sub-pages, as its tabs: this server, then each panel. */
function SubRows({ onNavigate }: { onNavigate?: () => void }) {
  const t = useT(shellMessages);
  const panels = (usePanels().data ?? []).filter((p) => p.id > 0);
  return (
    // The guide line sits under the parent's icon; sub-labels line up with its label.
    <ul className="ms-5 mt-0.5 flex flex-col gap-0.5 border-s border-kumo-hairline ps-2">
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
  const { pathname } = useLocation();
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
  // A row with sub-pages opens while one of them is shown; then the sub-row is the current one.
  const open = Boolean(item.perPanel) && pathname.startsWith(item.to);
  const Caret = open ? CaretDownIcon : CaretRightIcon;
  return (
    <>
      <NavLink
        to={item.to}
        end={item.to === '/' || item.perPanel}
        onClick={onNavigate}
        className={row}
      >
        <Icon size={16} aria-hidden="true" className="shrink-0 text-kumo-subtle" />
        <span>{label}</span>
        {item.perPanel ? (
          <Caret
            size={14}
            aria-hidden="true"
            className="ms-auto text-kumo-subtle rtl:-scale-x-100"
          />
        ) : null}
      </NavLink>
      {open ? <SubRows onNavigate={onNavigate} /> : null}
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
        <div key={group.label ?? i} className="flex flex-col gap-0.5">
          {group.label && !collapsed ? (
            <h2 className="px-3 pt-1 pb-0.5 text-sm text-kumo-subtle">{t(group.label)}</h2>
          ) : null}
          {group.label && collapsed ? (
            <hr aria-hidden="true" className="mx-2 my-1 border-kumo-hairline" />
          ) : null}
          <ul className="flex flex-col gap-0.5">
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
