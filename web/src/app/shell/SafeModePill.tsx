import { cn } from '@cloudflare/kumo/utils';
import { ShieldCheckIcon, ShieldWarningIcon } from '@phosphor-icons/react';
import { Link } from 'react-router';
import { useSafeMode } from '../../features/settings/use-safe-mode';
import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';

/** Safe Mode at a glance, where the dashboard shows the plan. Opens its settings. */
export function SafeModePill() {
  const t = useT(shellMessages);
  const safe = useSafeMode();
  if (!safe.data) return null;
  const on = safe.data.enabled;
  const label = on ? t('safeModeOn') : t('safeModeOff');
  const Icon = on ? ShieldCheckIcon : ShieldWarningIcon;
  return (
    <Link
      to="/settings/safe-mode"
      aria-label={label}
      title={label}
      className={cn(
        'inline-flex h-8 items-center gap-1.5 rounded-full px-2.5 text-sm ring hover:bg-kumo-tint pointer-coarse:h-11 pointer-coarse:min-w-11 pointer-coarse:justify-center',
        on
          ? 'text-kumo-default ring-kumo-line'
          : 'bg-kumo-warning-tint text-kumo-default ring-kumo-warning/40',
      )}
    >
      <Icon
        size={16}
        weight="bold"
        aria-hidden="true"
        className={on ? 'text-kumo-success' : 'text-kumo-warning'}
      />
      <span className="hidden sm:inline">{label}</span>
    </Link>
  );
}
