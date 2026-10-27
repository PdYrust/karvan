import { buttonVariants } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { InfoIcon, LinkSimpleIcon, UserIcon, WarningIcon } from '@phosphor-icons/react';
import { useId, type ReactNode } from 'react';
import { Link } from 'react-router';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { Impact, JobSummary } from '../../../lib/api/migrations';
import { stepPath } from '../steps';
import type { AdminPassword } from '../wizard-context';
import { startMessages } from './messages';
/** How the moved admin gets its password, and whether a chosen one is still missing. */
export function adminLine(
  summary: JobSummary,
  password: AdminPassword,
  valid: boolean,
): 'adminMerged' | 'adminTyped' | 'adminMissing' | 'adminKeep' | 'adminMade' {
  if (summary.newAdmin.merged) return 'adminMerged';
  if (password.choice === 'new') return valid ? 'adminTyped' : 'adminMissing';
  return summary.canKeepPassword ? 'adminKeep' : 'adminMade';
}
function Line({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <li className="flex items-start gap-2.5">
      <span className="mt-0.5 shrink-0 text-kumo-subtle">{icon}</span>
      <span>{children}</span>
    </li>
  );
}
/** A short flagged list of what the run changes for users and for the admin. */
export function UserEffects({
  jobId,
  impact,
  summary,
  admin,
}: {
  jobId: string;
  impact: Impact | undefined;
  summary: JobSummary;
  admin: ReturnType<typeof adminLine>;
}) {
  const t = useT(startMessages);
  const titleId = useId();
  const configs = impact?.configChanged ?? 0;
  const renamed = impact?.renamed ?? 0;
  const left = impact?.skipped ?? 0;
  const warn = <WarningIcon size={16} weight="bold" className="text-kumo-warning" />;
  return (
    <section
      aria-labelledby={titleId}
      className="flex flex-col gap-3 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-5"
    >
      <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
        {t('usersTitle')}
      </h2>
      <ul className="flex max-w-[720px] flex-col gap-2.5">
        <Line icon={<LinkSimpleIcon size={16} weight="bold" />}>
          {t('newLinks')} <span className="text-kumo-subtle">{t('oldLinks')}</span>
        </Line>
        <Line icon={configs > 0 ? warn : <InfoIcon size={16} weight="bold" />}>
          {configs > 0 ? t('configs', { count: configs }) : t('noConfigs')}
        </Line>
        {renamed > 0 ? (
          <Line icon={<InfoIcon size={16} weight="bold" />}>
            {t('renamed', { count: renamed })}
          </Line>
        ) : null}
        {left > 0 ? (
          <Line icon={<InfoIcon size={16} weight="bold" />}>{t('leftOut', { count: left })}</Line>
        ) : null}
        <Line icon={admin === 'adminMissing' ? warn : <UserIcon size={16} weight="bold" />}>
          <CodeText text={t(admin, { name: `\`${summary.newAdmin.username}\`` })} />
        </Line>
      </ul>
      {admin === 'adminMissing' ? (
        <div>
          <Link
            to={stepPath('compare', jobId)}
            className={cn(buttonVariants({ variant: 'secondary' }), 'pointer-coarse:h-9')}
          >
            {t('openCompare')}
          </Link>
        </div>
      ) : null}
    </section>
  );
}
