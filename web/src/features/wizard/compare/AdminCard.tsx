import { Button } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { Input } from '@cloudflare/kumo/components/input';
import { Radio } from '@cloudflare/kumo/components/radio';
import { useState, type FormEvent } from 'react';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { FormField } from '../../../components/form/FormField';
import { Section } from '../../../components/layout/Section';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { Conflict, JobSummary } from '../../../lib/api/migrations';
import { setAdmin, type AdminPlan } from '../../../lib/plan/plan';
import { useWizard } from '../wizard-context';
import { conflictMessages } from './conflict-messages';
import { compareMessages } from './messages';
import { PasswordChoice } from './PasswordChoice';
import { RoleFields } from './RoleFields';

type Mode = 'create' | 'merge';

/** A taken name, or a merge into a name the destination does not have, with the ways out. */
function NameProblem({ c01, c30 }: { c01: Conflict | undefined; c30: Conflict | undefined }) {
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
            className="pointer-coarse:h-11"
            onClick={() => plan.edit(setAdmin({ username: suggestion, mode: 'create' }))}
          >
            <CodeText text={t('useName', { name: suggestion })} />
          </Button>
        ) : null}
        <Button
          className="pointer-coarse:h-11"
          onClick={() => plan.edit(setAdmin({ username: name, mode: 'merge' }))}
        >
          <CodeText text={t('mergeInto', { name })} />
        </Button>
      </div>
    </div>
  );
}

/** How the admin lands, and under which name; the name saves when the operator says so. */
function NameFields({ admin, mode, name }: { admin: AdminPlan; mode: Mode; name: string }) {
  const t = useT(compareMessages);
  const { plan } = useWizard();
  const [draft, setDraft] = useState(name);
  const save = (e: FormEvent) => {
    e.preventDefault();
    plan.edit(setAdmin({ username: draft.trim(), mode }));
  };
  return (
    <>
      <Radio.Group
        legend={t('modeLabel')}
        orientation="horizontal"
        value={mode}
        onValueChange={(m: Mode) =>
          plan.edit(setAdmin({ username: admin.username || name, mode: m }))
        }
      >
        <Radio.Item value="create" label={t('modeCreate')} />
        <Radio.Item value="merge" label={t('modeMerge')} />
      </Radio.Group>
      <form onSubmit={save} className="flex flex-col gap-2">
        <FormField id="admin-name" label={t(mode === 'merge' ? 'mergeLabel' : 'usernameLabel')}>
          <div className="flex gap-2">
            <Input
              id="admin-name"
              aria-labelledby="admin-name-label"
              dir="ltr"
              autoComplete="off"
              spellCheck={false}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              className="w-full sm:w-72 pointer-coarse:h-11"
            />
            <Button type="submit" disabled={draft.trim() === name} className="pointer-coarse:h-11">
              {t('save')}
            </Button>
          </div>
        </FormField>
      </form>
    </>
  );
}

/** Step 3 starts with the admin: its name or a merge, its role, and its password. */
export function AdminCard({
  summary,
  conflicts,
}: {
  summary: JobSummary;
  conflicts: readonly Conflict[];
}) {
  const t = useT(compareMessages);
  const { plan } = useWizard();
  const admin = plan.plan?.admin ?? {};
  const outcome = summary.newAdmin;
  const mode: Mode = outcome.merged || admin.mode === 'merge' ? 'merge' : 'create';
  const name = admin.username || outcome.username || summary.admin;
  const destKind = summary.destKind || 'pasarguard';
  const withRole = destKind === 'pasarguard' && !outcome.merged;
  return (
    <Section title={t('adminTitle')}>
      <div className="flex flex-col gap-5 px-4 pb-5">
        <p className="text-kumo-subtle">{t('adminIntro')}</p>
        <NameProblem
          c01={conflicts.find((c) => c.type === 'C01')}
          c30={conflicts.find((c) => c.type === 'C30')}
        />
        {/* On a wide screen the role and what it allows sit next to the name and password. */}
        <div className={cn('grid gap-5', withRole && 'lg:grid-cols-2 lg:gap-8')}>
          <div className="flex min-w-0 flex-col gap-5">
            <NameFields key={`${mode}:${name}`} admin={admin} mode={mode} name={name} />
            {outcome.merged ? (
              <InlineAlert tone="info">
                <CodeText text={t('merged', { name: outcome.username })} />
              </InlineAlert>
            ) : (
              <PasswordChoice
                canKeep={summary.canKeepPassword}
                username={outcome.username}
                kind={destKind}
              />
            )}
          </div>
          {withRole ? (
            <div className="min-w-0">
              <RoleFields
                roleId={admin.roleId || outcome.roleId}
                conflicts={conflicts}
                users={summary.users}
              />
            </div>
          ) : null}
        </div>
      </div>
    </Section>
  );
}
