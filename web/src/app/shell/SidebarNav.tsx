import { Tooltip } from '@cloudflare/kumo/components/tooltip';
import { cn } from '@cloudflare/kumo/utils';
import { NavLink } from 'react-router';
import { useLocale, useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { shellMessages } from './messages';
import { nav, type NavItem } from './nav';

// Hover colors change at once (Kumo's rule), so no transition here.
const row =
  'flex min-h-9 items-center gap-3 rounded-md px-3 text-base pointer-coarse:min-h-11 text-kumo-default hover:bg-kumo-tint aria-[current=page]:bg-kumo-tint aria-[current=page]:font-medium aria-[current=page]:text-kumo-strong';
const iconRow =
  'grid size-10 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint aria-[current=page]:bg-kumo-tint pointer-coarse:size-11';

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
  const ts = useT(sharedMessages);
  const { dir } = useLocale();
  const Icon = item.icon;
  const label = t(item.label);

  if (!item.to) {
    const soon = (
      <span
        aria-disabled="true"
        aria-label={collapsed ? `${label} (${ts('soon')})` : undefined}
        className={cn(
          collapsed ? iconRow : row,
          'cursor-default text-kumo-subtle hover:bg-transparent',
        )}
      >
        <Icon size={18} aria-hidden="true" className="text-kumo-inactive" />
        {collapsed ? null : (
          <>
            <span className="flex-1">{label}</span>
            <span className="rounded-md px-1.5 text-tiny text-kumo-subtle ring ring-kumo-line">
              {ts('soon')}
            </span>
          </>
        )}
      </span>
    );
    return collapsed ? (
      <Tooltip content={`${label} (${ts('soon')})`} side={dir === 'rtl' ? 'left' : 'right'} asChild>
        {soon}
      </Tooltip>
    ) : (
      soon
    );
  }

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
  return (
    <NavLink to={item.to} end={item.to === '/'} onClick={onNavigate} className={row}>
      <Icon size={18} aria-hidden="true" className="text-kumo-subtle" />
      <span>{label}</span>
    </NavLink>
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
      className={cn('flex flex-col gap-4', collapsed && 'items-center gap-2')}
    >
      {nav.map((group, i) => (
        <div key={group.label ?? i} className="flex flex-col gap-0.5">
          {group.label && !collapsed ? (
            <h2 className="px-3 pt-1 pb-1 text-sm text-kumo-subtle">{t(group.label)}</h2>
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
