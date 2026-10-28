import { Select } from '@cloudflare/kumo/components/select';
import { useQuery } from '@tanstack/react-query';
import { useId } from 'react';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { FormField } from '../../../components/form/FormField';
import { FactList, type Fact } from '../../../components/layout/FactList';
import { useT } from '../../../i18n/locale';
import { getRoles, migrationKeys, type Conflict, type Role } from '../../../lib/api/migrations';
import type { Translate } from '../../../i18n/core';
import { setAdmin } from '../../../lib/plan/plan';
import { useFormat } from '../../../lib/preferences/region';
import { useErrorText } from '../../../lib/errors/errors';
import { useWizard } from '../wizard-context';
import { conflictMessages, type ConflictKey } from './conflict-messages';
import { compareMessages, type CompareKey } from './messages';
import { RoleGroups } from './RoleGroups';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

/** The role warnings of the whole job (api-migration.md section 4). */
const roleConflicts = ['C09', 'C33', 'C34', 'C35', 'C36', 'C37'];

/** The roles as Select items, an owner role marked. */
export function roleItems(roles: readonly Role[], t: Translate<CompareKey>) {
  return Object.fromEntries(
    roles.map((r) => [String(r.id), r.isOwner ? t('roleOwner', { name: r.name }) : r.name]),
  );
}

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

/**
 * The PasarGuard role one admin gets, first on its card: a picker, or the
 * existing admin's own role as text on a merge (D33); then its warnings,
 * the groups it gives users and what it allows.
 */
export function RoleFields({
  source,
  roleId,
  fixed,
  conflicts,
  users,
}: {
  /** The admin's name on the old panel. */
  source: string;
  roleId: number;
  fixed: boolean;
  conflicts: readonly Conflict[];
  users: number;
}) {
  const t = useT(compareMessages);
  const tc = useT(conflictMessages);
  const fmt = useFormat();
  const errorText = useErrorText();
  const id = useId();
  const { jobId, plan } = useWizard();
  const roles = useQuery({ queryKey: migrationKeys.roles(jobId), queryFn: () => getRoles(jobId) });
  const load = useFirstLoad(roles.data);
  if (roles.isPending || load.holding) return <Bone className="h-48 w-full" />;
  if (roles.isError) return <InlineAlert>{errorText(roles.error)}</InlineAlert>;
  const role = roles.data.roles.find((r) => r.id === roleId);
  const warnings = conflicts.filter((c) => roleConflicts.includes(c.type));
  return (
    <Reveal on={load.reveal}>
      <div className="flex flex-col gap-4">
        {fixed ? (
          <div className="flex flex-col gap-1">
            <span className="text-sm font-medium text-kumo-strong">{t('roleLabel')}</span>
            <bdi className="text-kumo-strong">{role?.name ?? String(roleId)}</bdi>
            <span className="text-sm text-kumo-subtle">{t('roleFixedHint')}</span>
          </div>
        ) : (
          <FormField id={id} label={t('roleLabel')}>
            <Select
              id={id}
              aria-labelledby={`${id}-label`}
              className="w-full sm:w-72 pointer-coarse:h-9"
              value={String(roleId)}
              items={roleItems(roles.data.roles, t)}
              onValueChange={(v) => v && plan.edit(setAdmin(source, { roleId: Number(v) }))}
            />
          </FormField>
        )}
        {warnings.map((c) => (
          <InlineAlert key={c.type} tone={c.severity === 'blocker' ? 'danger' : 'warning'}>
            {tc(`job_${c.type}` as ConflictKey, {
              max: fmt.number(Number(c.detail?.max ?? 0)),
              users: fmt.number(Number(c.detail?.users ?? users)),
            })}
          </InlineAlert>
        ))}
        {role ? <RoleGroups role={role} /> : null}
        {role ? <RoleFacts role={role} /> : null}
      </div>
    </Reveal>
  );
}
