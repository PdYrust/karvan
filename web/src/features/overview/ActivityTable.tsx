import { cn } from '@cloudflare/kumo/utils';
import {
  KeyIcon,
  LockKeyIcon,
  SignInIcon,
  SignOutIcon,
  TerminalWindowIcon,
  WarningCircleIcon,
  type Icon,
} from '@phosphor-icons/react';
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

const cell = 'px-4 py-3 align-middle';

function Row({ entry, now }: { entry: ActivityEntry; now: Date }) {
  const t = useT(overviewMessages);
  const { locale } = useLocale();
  const failed = entry.outcome === 'failure' || entry.action === 'lockout';
  const Icon = failed ? WarningCircleIcon : (icons[entry.action] ?? SignInIcon);
  const key = messageKey(entry);
  const at = new Date(entry.at);
  const from =
    entry.actor === 'cli' ? (
      <span>{t('fromCommandLine')}</span>
    ) : (
      <bdi dir="ltr" className="font-mono text-tiny">
        {entry.ip || '—'}
      </bdi>
    );
  return (
    <tr className="border-t border-kumo-hairline">
      <td className={cell}>
        <span className="flex items-center gap-3">
          <Icon
            size={18}
            aria-hidden="true"
            className={cn('shrink-0', failed ? 'text-kumo-warning' : 'text-kumo-subtle')}
          />
          <span className="flex flex-col">
            <span className="text-kumo-default">
              {key ? t(key) : t('act.unknown', { action: entry.action })}
            </span>
            <span className="text-sm text-kumo-subtle sm:hidden">{from}</span>
          </span>
        </span>
      </td>
      <td className={cn(cell, 'hidden text-kumo-subtle sm:table-cell')}>{from}</td>
      <td className={cn(cell, 'text-end whitespace-nowrap text-kumo-subtle')}>
        <time dateTime={entry.at} title={formatDateTime(at, locale)}>
          {formatRelative(at, now, locale)}
        </time>
      </td>
    </tr>
  );
}

export function ActivityTable({ entries, now }: { entries: ActivityEntry[]; now: Date }) {
  const t = useT(overviewMessages);
  return (
    <Section title={t('activityTitle')}>
      {entries.length === 0 ? (
        <p className="px-4 pt-2 pb-6 text-kumo-subtle">{t('activityEmpty')}</p>
      ) : (
        <>
          <table className="mt-2 w-full border-collapse text-start">
            <thead className="bg-kumo-elevated text-sm text-kumo-subtle">
              <tr className="border-t border-kumo-hairline">
                <th scope="col" className={cn(cell, 'text-start font-medium')}>
                  {t('colEvent')}
                </th>
                <th scope="col" className={cn(cell, 'hidden text-start font-medium sm:table-cell')}>
                  {t('colFrom')}
                </th>
                <th scope="col" className={cn(cell, 'text-end font-medium')}>
                  {t('colWhen')}
                </th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <Row key={e.id} entry={e} now={now} />
              ))}
            </tbody>
          </table>
          <p className="border-t border-kumo-hairline px-4 py-3 text-sm text-kumo-subtle">
            {t('activityShowing', { count: entries.length })}
          </p>
        </>
      )}
    </Section>
  );
}
