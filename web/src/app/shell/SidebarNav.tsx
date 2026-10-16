import { cn } from '@cloudflare/kumo/utils';
import { NavLink } from 'react-router';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { shellMessages } from './messages';
import { nav, type NavItem } from './nav';

const row =
  'flex min-h-9 items-center gap-3 rounded-md px-3 text-base transition-colors pointer-coarse:min-h-11';

function NavRow({ item, onNavigate }: { item: NavItem; onNavigate?: () => void }) {
  const t = useT(shellMessages);
  const ts = useT(sharedMessages);
  const Icon = item.icon;
  if (!item.to) {
    return (
      <span aria-disabled="true" className={cn(row, 'cursor-default text-kumo-subtle')}>
        <Icon size={18} aria-hidden="true" className="text-kumo-inactive" />
        <span className="flex-1">{t(item.label)}</span>
        <span className="rounded-md px-1.5 text-tiny text-kumo-subtle ring ring-kumo-line">
          {ts('soon')}
        </span>
      </span>
    );
  }
  return (
    <NavLink
      to={item.to}
      end
      onClick={onNavigate}
      className={({ isActive }) =>
        cn(
          row,
          isActive
            ? 'bg-kumo-tint font-medium text-kumo-strong'
            : 'text-kumo-default hover:bg-kumo-tint',
        )
      }
    >
      <Icon size={18} aria-hidden="true" className="text-kumo-subtle" />
      <span>{t(item.label)}</span>
    </NavLink>
  );
}

export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const t = useT(shellMessages);
  return (
    <nav aria-label={t('mainNav')} className="flex flex-col gap-4">
      {nav.map((group, i) => (
        <div key={group.label ?? i} className="flex flex-col gap-1">
          {group.label ? (
            <h2 className="px-3 pb-1 text-sm text-kumo-subtle">{t(group.label)}</h2>
          ) : null}
          <ul className="flex flex-col gap-1">
            {group.items.map((item) => (
              <li key={item.label}>
                <NavRow item={item} onNavigate={onNavigate} />
              </li>
            ))}
          </ul>
        </div>
      ))}
    </nav>
  );
}
