import {
  KeyIcon,
  LockKeyIcon,
  SignInIcon,
  SignOutIcon,
  TerminalWindowIcon,
  WarningCircleIcon,
  type Icon,
} from '@phosphor-icons/react';
import { cn } from '@cloudflare/kumo/utils';
import { Section } from '../../components/layout/Section';
import { useLocale, useT } from '../../i18n/locale';
import type { ActivityEntry } from '../../lib/api/overview';
import { formatDateTime, formatRelative } from '../../lib/format/date';
import { overviewMessages, type OverviewKey } from './messages';

const icons: Record<string, Icon> = {
  login: SignInIcon,
  logout: SignOutIcon,
  lockout: LockKeyIcon,
  'totp.enable': KeyIcon,
  'totp.disable': KeyIcon,
  'cli.init': TerminalWindowIcon,
  'cli.reset': TerminalWindowIcon,
};

function messageKey(e: ActivityEntry): OverviewKey | null {
  const key = `act.${e.action}.${e.outcome}`;
  return key in overviewMessages.en ? (key as OverviewKey) : null;
}

function Row({ entry, now }: { entry: ActivityEntry; now: Date }) {
  const t = useT(overviewMessages);
  const { locale } = useLocale();
  const failed = entry.outcome === 'failure' || entry.action === 'lockout';
  const Icon = failed ? WarningCircleIcon : (icons[entry.action] ?? SignInIcon);
  const key = messageKey(entry);
  const at = new Date(entry.at);
  return (
    <li className="flex items-start gap-3 px-4 py-3 sm:px-5">
      <span
        className={cn(
          'mt-0.5 grid size-8 shrink-0 place-items-center rounded-lg',
          failed ? 'bg-kumo-warning-tint text-kumo-warning' : 'bg-kumo-recessed text-kumo-subtle',
        )}
      >
        <Icon size={18} aria-hidden="true" />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5 sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
        <p className="text-kumo-default">
          {key ? t(key) : t('act.unknown', { action: entry.action })}
        </p>
        <p className="flex shrink-0 items-center gap-2 text-sm text-kumo-subtle">
          {entry.actor === 'cli' ? (
            <span>{t('fromCommandLine')}</span>
          ) : (
            <bdi dir="ltr" className="font-mono text-tiny">
              {entry.ip}
            </bdi>
          )}
          <time dateTime={entry.at} title={formatDateTime(at, locale)}>
            {formatRelative(at, now, locale)}
          </time>
        </p>
      </div>
    </li>
  );
}

export function ActivitySection({
  entries,
  now,
  className,
}: {
  entries: ActivityEntry[];
  now: Date;
  className?: string;
}) {
  const t = useT(overviewMessages);
  return (
    <Section title={t('activityTitle')} className={className}>
      {entries.length === 0 ? (
        <p className="px-5 py-8 text-center text-kumo-subtle">{t('activityEmpty')}</p>
      ) : (
        <ol className="divide-y divide-kumo-hairline">
          {entries.map((e) => (
            <Row key={e.id} entry={e} now={now} />
          ))}
        </ol>
      )}
    </Section>
  );
}
