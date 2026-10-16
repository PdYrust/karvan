import { cn } from '@cloudflare/kumo/utils';
import { NavLink } from 'react-router';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { shellMessages } from './messages';
import { nav, type NavItem } from './nav';

const row =
  'relative flex min-h-10 items-center gap-3 rounded-lg px-3 text-base pointer-coarse:min-h-11';

function NavRow({ item, onNavigate }: { item: NavItem; onNavigate?: () => void }) {
  const t = useT(shellMessages);
  const ts = useT(sharedMessages);
  const Icon = item.icon;
  if (!item.to) {
    return (
      <span aria-disabled="true" className={cn(row, 'cursor-default text-kumo-inactive')}>
        <Icon size={20} aria-hidden="true" />
        <span className="flex-1">{t(item.label)}</span>
        <span className="rounded-md bg-kumo-recessed px-1.5 text-tiny text-kumo-subtle">
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
          'transition-colors',
          isActive
            ? 'bg-kumo-tint font-semibold text-kumo-strong'
            : 'text-kumo-default hover:bg-kumo-tint/60',
        )
      }
    >
      {({ isActive }) => (
        <>
          {/* The current page is marked with a short stretch of road. */}
          {isActive ? (
            <span
              aria-hidden="true"
              className="absolute inset-y-2 start-0 w-[3px] rounded-full bg-[repeating-linear-gradient(to_bottom,var(--color-karvan-saffron)_0_5px,transparent_5px_8px)]"
            />
          ) : null}
          <Icon size={20} weight={isActive ? 'bold' : 'regular'} aria-hidden="true" />
          <span>{t(item.label)}</span>
        </>
      )}
    </NavLink>
  );
}

export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const t = useT(shellMessages);
  return (
    <nav aria-label={t('mainNav')} className="flex flex-col gap-5">
      {nav.map((group, i) => (
        <div key={group.label ?? i} className="flex flex-col gap-0.5">
          {group.label ? (
            <h2 className="px-3 pb-1 text-sm text-kumo-subtle">{t(group.label)}</h2>
          ) : null}
          <ul className="flex flex-col gap-0.5">
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
