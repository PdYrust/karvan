import { Select } from '@cloudflare/kumo/components/select';
import { useQuery } from '@tanstack/react-query';
import { useId } from 'react';
import { FormField } from '../../../components/form/FormField';
import { Section } from '../../../components/layout/Section';
import { useT } from '../../../i18n/locale';
import {
  getRoles,
  migrationKeys,
  type AdminInfo,
  type Conflict,
  type JobSummary,
} from '../../../lib/api/migrations';
import { setRoleForAll } from '../../../lib/plan/plan';
import { useWizard } from '../wizard-context';
import { AdminCard } from './AdminCard';
import { roleChoosers, sharedRole } from './admins';
import { compareMessages } from './messages';
import { roleItems } from './RoleFields';

/** "Same role for all": one pick sets the role of every admin whose role can change. */
function SameRole({ admins }: { admins: readonly AdminInfo[] }) {
  const t = useT(compareMessages);
  const id = useId();
  const { jobId, plan } = useWizard();
  const roles = useQuery({ queryKey: migrationKeys.roles(jobId), queryFn: () => getRoles(jobId) });
  if (!roles.data) return null;
  // The roles as the screen shows them, edits still on their way included.
  const shown = admins.map((a) => ({
    ...a,
    roleId: plan.plan?.admins?.[a.username]?.roleId || a.roleId,
  }));
  const shared = sharedRole(shown);
  return (
    <Section title={t('sameRoleTitle')}>
      <div className="px-4 pb-4">
        <FormField id={id} label={t('roleLabel')} hint={t('sameRoleIntro')}>
          <Select
            id={id}
            aria-labelledby={`${id}-label`}
            aria-describedby={`${id}-hint`}
            className="w-full sm:w-72 pointer-coarse:h-9"
            value={shared === null ? null : String(shared)}
            placeholder={t('sameRoleMixed')}
            items={roleItems(roles.data.roles, t)}
            onValueChange={(v) =>
              v &&
              plan.edit(
                setRoleForAll(
                  admins.map((a) => a.username),
                  Number(v),
                ),
              )
            }
          />
        </FormField>
      </div>
    </Section>
  );
}

/** Step 3 starts with the admins: one card per admin that moves, "Same role for all" above them. */
export function AdminList({
  summary,
  conflicts,
}: {
  summary: JobSummary;
  conflicts: readonly Conflict[];
}) {
  const moving = (summary.admins ?? []).filter((a) => a.moves);
  const choosers = roleChoosers(moving);
  const pg = (summary.destKind || 'pasarguard') === 'pasarguard';
  return (
    <>
      {pg && choosers.length > 1 ? <SameRole admins={choosers} /> : null}
      {moving.map((a) => (
        <AdminCard
          key={a.username}
          summary={summary}
          admin={a}
          conflicts={conflicts}
          several={moving.length > 1}
        />
      ))}
    </>
  );
}
