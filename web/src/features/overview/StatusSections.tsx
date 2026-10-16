import { Badge } from '@cloudflare/kumo/components/badge';
import { CheckCircleIcon, ShieldCheckIcon, WarningIcon } from '@phosphor-icons/react';
import { Section } from '../../components/layout/Section';
import { useLocale, useT } from '../../i18n/locale';
import type { Overview } from '../../lib/api/overview';
import { formatDateTime, formatRelative } from '../../lib/format/date';
import { overviewMessages } from './messages';

export function SafeModeSection({ on }: { on: boolean }) {
  const t = useT(overviewMessages);
  return (
    <Section
      title={t('safeModeTitle')}
      actions={
        <Badge
          variant={on ? 'success' : 'warning'}
          icon={on ? <CheckCircleIcon size={14} /> : <WarningIcon size={14} />}
        >
          {on ? t('on') : t('off')}
        </Badge>
      }
    >
      <div className="flex gap-3 px-4 py-4 sm:px-5">
        <ShieldCheckIcon
          size={22}
          aria-hidden="true"
          className="mt-0.5 shrink-0 text-karvan-firouzeh"
        />
        <p className="text-kumo-subtle">{t('safeModeBody')}</p>
      </div>
    </Section>
  );
}

export function SecuritySection({ security, now }: { security: Overview['security']; now: Date }) {
  const t = useT(overviewMessages);
  const { locale } = useLocale();
  const prev = security.previousSignIn;
  return (
    <Section title={t('securityTitle')}>
      <div className="divide-y divide-kumo-hairline">
        <div className="flex flex-col gap-1 px-4 py-4 sm:px-5">
          <div className="flex items-center justify-between gap-3">
            <h3 className="font-medium text-kumo-default">{t('twoStep')}</h3>
            <Badge
              variant={security.totpEnabled ? 'success' : 'warning'}
              icon={
                security.totpEnabled ? <CheckCircleIcon size={14} /> : <WarningIcon size={14} />
              }
            >
              {security.totpEnabled ? t('on') : t('off')}
            </Badge>
          </div>
          <p className="text-sm text-kumo-subtle">
            {security.totpEnabled ? t('twoStepOnHint') : t('twoStepOffHint')}
          </p>
        </div>
        <div className="flex flex-col gap-1 px-4 py-4 sm:px-5">
          <h3 className="font-medium text-kumo-default">{t('previousSignIn')}</h3>
          <p className="text-sm text-kumo-subtle">
            {prev ? (
              <>
                <time dateTime={prev.at} title={formatDateTime(new Date(prev.at), locale)}>
                  {t('previousSignInValue', {
                    when: formatRelative(new Date(prev.at), now, locale),
                  })}
                </time>{' '}
                <bdi dir="ltr" className="font-mono text-tiny text-kumo-default">
                  {prev.ip || '—'}
                </bdi>
              </>
            ) : (
              t('firstSignIn')
            )}
          </p>
        </div>
      </div>
    </Section>
  );
}
