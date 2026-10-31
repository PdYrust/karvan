import { Badge } from '@cloudflare/kumo/components/badge';
import { Button } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { Input } from '@cloudflare/kumo/components/input';
import { Radio } from '@cloudflare/kumo/components/radio';
import { useId, useState, type FormEvent } from 'react';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { FormField } from '../../../components/form/FormField';
import { Section } from '../../../components/layout/Section';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { AdminInfo, Conflict, JobSummary } from '../../../lib/api/migrations';
import { setAdmin } from '../../../lib/plan/plan';
import { useWizard } from '../wizard-context';
import { adminConflicts } from './admins';
import { conflictMessages } from './conflict-messages';
import { compareMessages } from './messages';
import { PasswordChoice } from './PasswordChoice';
import { RoleFields } from './RoleFields';

type Mode = 'create' | 'merge';

/** A taken name, or a merge into a name the destination does not have, with the ways out. */
function NameProblem({
  source,
  c01,
  c30,
}: {
  source: string;
  c01: Conflict | undefined;
  c30: Conflict | undefined;
}) {
  const t = useT(compareMessages);
  const tc = useT(conflictMessages);
  const { plan } = useWizard();
  if (c30) return <InlineAlert>{tc('job_C30')}</InlineAlert>;
  if (!c01) return null;
  const name = c01.detail?.username ?? '';
  if (c01.detail?.missing)
    return (
      <InlineAlert>
        <CodeText text={t('adminMissing', { name })} />
      </InlineAlert>
    );
  const suggestion = c01.detail?.suggestion;
  return (
    <div className="flex flex-col gap-3">
      <InlineAlert tone="warning">
        <CodeText text={t('adminTaken', { name })} />
      </InlineAlert>
      <div className="flex flex-wrap gap-2">
        {suggestion ? (
          <Button
            className="pointer-coarse:h-9"
            onClick={() => plan.edit(setAdmin(source, { username: suggestion, mode: 'create' }))}
          >
            <CodeText text={t('useName', { name: suggestion })} />
          </Button>
        ) : null}
        <Button
          className="pointer-coarse:h-9"
          onClick={() => plan.edit(setAdmin(source, { username: name, mode: 'merge' }))}
        >
          <CodeText text={t('mergeInto', { name })} />
        </Button>
      </div>
    </div>
  );
}

/** How the admin lands, and under which name; the name saves when the operator says so. */
function NameFields({ source, mode, name }: { source: string; mode: Mode; name: string }) {
  const t = useT(compareMessages);
  const { plan } = useWizard();
  const id = useId();
  const [draft, setDraft] = useState(name);
  const save = (e: FormEvent) => {
    e.preventDefault();
    plan.edit(setAdmin(source, { username: draft.trim(), mode }));
  };
  return (
    <>
      <Radio.Group
        legend={t('modeLabel')}
        orientation="horizontal"
        value={mode}
        onValueChange={(m: Mode) => plan.edit(setAdmin(source, { username: name, mode: m }))}
      >
        <Radio.Item value="create" label={t('modeCreate')} />
        <Radio.Item value="merge" label={t('modeMerge')} />
      </Radio.Group>
      <form onSubmit={save} className="flex flex-col gap-2">
        <FormField id={id} label={t(mode === 'merge' ? 'mergeLabel' : 'usernameLabel')}>
          <div className="flex gap-2">
            <Input
              id={id}
              aria-labelledby={`${id}-label`}
              dir="ltr"
              autoComplete="off"
              spellCheck={false}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              className="w-full sm:w-72 pointer-coarse:h-9"
            />
            <Button type="submit" disabled={draft.trim() === name} className="pointer-coarse:h-9">
              {t('save')}
            </Button>
          </div>
        </FormField>
      </form>
    </>
  );
}

/** An admin other than the login's always gets a password Karvan makes. */
function MadePassword() {
  const t = useT(compareMessages);
  return (
    <div className="flex flex-col gap-1">
      <span className="text-base font-medium text-kumo-default">{t('passwordLabel')}</span>
      <span>{t('makeOne')}</span>
      <span className="text-sm text-kumo-subtle">{t('makeOneHint')}</span>
    </div>
  );
}

/**
 * One admin of the old panel and how it lands: on PasarGuard its role
 * first, with the groups it gives users; then its name or a merge, and its
 * password.
 */
export function AdminCard({
  summary,
  admin,
  conflicts,
  several,
}: {
  summary: JobSummary;
  admin: AdminInfo;
  conflicts: readonly Conflict[];
  /** The job moves more than one admin, so each card names its own. */
  several: boolean;
}) {
  const t = useT(compareMessages);
  const tc = useT(conflictMessages);
  const { plan } = useWizard();
  const source = admin.username;
  const entry = plan.plan?.admins?.[source] ?? {};
  const outcome = admin.newAdmin;
  const mode: Mode = outcome.merged || entry.mode === 'merge' ? 'merge' : 'create';
  const name = entry.username || outcome.username || source;
  const destKind = summary.destKind || 'pasarguard';
  const pg = destKind === 'pasarguard';
  const mine = adminConflicts(conflicts, admin);
  const c09 = mine.find((c) => c.type === 'C09');
  return (
    <Section
      title={several ? <CodeText text={t('adminTitleOf', { name: source })} /> : t('adminTitle')}
      actions={several && admin.isLogin ? <Badge variant="outline">{t('loginAdmin')}</Badge> : null}
    >
      <div className="flex flex-col gap-5 px-4 pb-5">
        <p className="text-kumo-subtle">{t('adminUsers', { count: admin.users })}</p>
        <NameProblem
          source={source}
          c01={mine.find((c) => c.type === 'C01')}
          c30={mine.find((c) => c.type === 'C30')}
        />
        {!pg && c09 ? <InlineAlert>{tc('job_C09')}</InlineAlert> : null}
        {/* On a wide screen the role and its groups sit before the name and the password. */}
        <div className={cn('grid gap-5', pg && 'lg:grid-cols-2 lg:gap-8')}>
          {pg ? (
            <div className="min-w-0">
              <RoleFields
                source={source}
                roleId={(!admin.roleFixed && entry.roleId) || admin.roleId}
                fixed={admin.roleFixed}
                conflicts={mine}
                users={admin.users}
              />
            </div>
          ) : null}
          <div className="flex min-w-0 flex-col gap-5">
            <NameFields key={`${mode}:${name}`} source={source} mode={mode} name={name} />
            {outcome.merged ? (
              <InlineAlert tone="info">
                <CodeText text={t('merged', { name: outcome.username })} />
              </InlineAlert>
            ) : admin.isLogin ? (
              <PasswordChoice
                canKeep={summary.canKeepPassword}
                username={outcome.username}
                kind={destKind}
              />
            ) : (
              <MadePassword />
            )}
          </div>
        </div>
      </div>
    </Section>
  );
}
