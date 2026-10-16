import { Badge } from '@cloudflare/kumo/components/badge';
import { CheckCircleIcon, WarningIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { useLocale, useT } from '../../i18n/locale';
import type { Overview } from '../../lib/api/overview';
import { formatDateTime, formatRelative } from '../../lib/format/date';
import { overviewMessages } from './messages';

function OnOff({ on }: { on: boolean }) {
  const t = useT(overviewMessages);
  return (
    <Badge
      variant={on ? 'success' : 'warning'}
      icon={on ? <CheckCircleIcon size={14} /> : <WarningIcon size={14} />}
    >
      {on ? t('on') : t('off')}
    </Badge>
  );
}

function Block({
  title,
  badge,
  children,
}: {
  title: string;
  badge?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-2 border-b border-kumo-hairline py-5 first:pt-0 last:border-b-0">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-heading font-semibold text-kumo-strong">{title}</h2>
        {badge}
      </div>
      {children}
    </section>
  );
}

/** The side column, like the summary column on a dashboard detail page. */
export function OverviewAside({
  data,
  username,
  now,
}: {
  data: Overview;
  username: string;
  now: Date;
}) {
  const t = useT(overviewMessages);
  const { locale } = useLocale();
  const prev = data.security.previousSignIn;
  return (
    <div className="flex flex-col">
      <Block title={t('safeModeTitle')} badge={<OnOff on={data.safeMode} />}>
        <p className="text-sm text-kumo-subtle">{t('safeModeBody')}</p>
      </Block>
      <Block title={t('twoStep')} badge={<OnOff on={data.security.totpEnabled} />}>
        <p className="text-sm text-kumo-subtle">
          {data.security.totpEnabled ? t('twoStepOnHint') : t('twoStepOffHint')}
        </p>
      </Block>
      <Block title={t('accountTitle')}>
        <dl className="flex flex-col text-sm">
          <div className="flex justify-between gap-4 border-b border-kumo-hairline py-2">
            <dt className="text-kumo-subtle">{t('signedInAs')}</dt>
            <dd>
              <bdi dir="ltr" className="font-mono text-tiny text-kumo-default">
                {username}
              </bdi>
            </dd>
          </div>
          <div className="flex justify-between gap-4 py-2">
            <dt className="text-kumo-subtle">{t('previousSignIn')}</dt>
            <dd className="text-end text-kumo-default">
              {prev ? (
                <>
                  <time dateTime={prev.at} title={formatDateTime(new Date(prev.at), locale)}>
                    {formatRelative(new Date(prev.at), now, locale)}
                  </time>
                  <bdi dir="ltr" className="block font-mono text-tiny text-kumo-subtle">
                    {prev.ip || '—'}
                  </bdi>
                </>
              ) : (
                t('firstSignIn')
              )}
            </dd>
          </div>
        </dl>
      </Block>
    </div>
  );
}
