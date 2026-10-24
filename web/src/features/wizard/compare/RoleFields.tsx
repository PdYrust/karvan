import { Select } from '@cloudflare/kumo/components/select';
import { useQuery } from '@tanstack/react-query';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { FormField } from '../../../components/form/FormField';
import { FactList, type Fact } from '../../../components/layout/FactList';
import { useT } from '../../../i18n/locale';
import { getRoles, migrationKeys, type Conflict, type Role } from '../../../lib/api/migrations';
import { setAdmin } from '../../../lib/plan/plan';
import { useFormat } from '../../../lib/preferences/region';
import { useErrorText } from '../../../lib/errors/errors';
import { useWizard } from '../wizard-context';
import { conflictMessages, type ConflictKey } from './conflict-messages';
import { compareMessages } from './messages';

/** The role warnings of the whole job (api-migration.md section 4). */
const roleConflicts = ['C33', 'C34', 'C35', 'C36', 'C37'];

function useRange() {
  const t = useT(compareMessages);
  return (
    min: number | null | undefined,
    max: number | null | undefined,
    f: (n: number) => string,
  ) => {
    const lo = min ?? null;
    const hi = max ?? null;
    if (lo !== null && hi !== null) return t('between', { min: f(lo), max: f(hi) });
    if (hi !== null) return t('upTo', { max: f(hi) });
    if (lo !== null) return t('atLeast', { min: f(lo) });
    return t('noLimit');
  };
}

function RoleFacts({ role }: { role: Role }) {
  const t = useT(compareMessages);
  const fmt = useFormat();
  const range = useRange();
  const l = role.limits;
  const facts: Fact[] = [
    {
      label: t('roleGroups'),
      value: role.allGroups ? (
        t('roleAllGroups')
      ) : role.groups.length ? (
        <bdi>{role.groups.map((g) => g.name).join(', ')}</bdi>
      ) : (
        t('roleNoGroups')
      ),
    },
    { label: t('roleUsers'), value: range(null, l.max_users, (n) => fmt.number(n)) },
    { label: t('roleData'), value: range(l.data_limit_min, l.data_limit_max, fmt.bytes) },
    { label: t('roleExpiry'), value: range(l.expire_min, l.expire_max, (s) => fmt.duration(s, 2)) },
    {
      label: t('roleHwid'),
      value: range(l.min_hwid_per_user, l.max_hwid_per_user, (n) => fmt.number(n)),
    },
    {
      label: t('roleReset'),
      value: t(role.features.can_use_reset_strategy ? 'allowed' : 'notAllowed'),
    },
    {
      label: t('roleNextPlan'),
      value: t(role.features.can_use_next_plan ? 'allowed' : 'notAllowed'),
    },
  ];
  if (role.requireTemplate) facts.push({ label: t('roleTemplates'), value: t('yes') });
  return <FactList facts={facts} />;
}

/** The PasarGuard role the moved admin gets: what it allows, and the warnings it brings. */
export function RoleFields({
  roleId,
  conflicts,
  users,
}: {
  roleId: number;
  conflicts: readonly Conflict[];
  users: number;
}) {
  const t = useT(compareMessages);
  const tc = useT(conflictMessages);
  const fmt = useFormat();
  const errorText = useErrorText();
  const { jobId, plan } = useWizard();
  const roles = useQuery({ queryKey: migrationKeys.roles(jobId), queryFn: () => getRoles(jobId) });
  if (roles.isPending) return <Bone className="h-48 w-full" />;
  if (roles.isError) return <InlineAlert>{errorText(roles.error)}</InlineAlert>;
  const current = roleId || roles.data.roleId;
  const role = roles.data.roles.find((r) => r.id === current);
  const items = Object.fromEntries(
    roles.data.roles.map((r) => [
      String(r.id),
      r.isOwner ? t('roleOwner', { name: r.name }) : r.name,
    ]),
  );
  const warnings = conflicts.filter((c) => roleConflicts.includes(c.type));
  return (
    <div className="flex flex-col gap-3">
      <FormField id="admin-role" label={t('roleLabel')}>
        <Select
          id="admin-role"
          aria-labelledby="admin-role-label"
          className="w-full sm:w-72 pointer-coarse:h-11"
          value={String(current)}
          items={items}
          onValueChange={(v) => v && plan.edit(setAdmin({ roleId: Number(v) }))}
        />
      </FormField>
      {warnings.map((c) => (
        <InlineAlert key={c.type} tone={c.severity === 'blocker' ? 'danger' : 'warning'}>
          {tc(`job_${c.type}` as ConflictKey, {
            max: fmt.number(Number(c.detail?.max ?? 0)),
            users: fmt.number(Number(c.detail?.users ?? users)),
          })}
        </InlineAlert>
      ))}
      {role ? <RoleFacts role={role} /> : null}
    </div>
  );
}
