import { Button } from '@cloudflare/kumo/components/button';
import { CaretDownIcon } from '@phosphor-icons/react';
import { useId, useState } from 'react';
import { StatusBadge, type StatusTone } from '../../../components/feedback/StatusBadge';
import { Section } from '../../../components/layout/Section';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { PanelKind } from '../../../lib/api/panels';
import type { ProblemGroup } from '../../../lib/api/migrations';
import { useWizard } from '../wizard-context';
import { conflictMessages } from './conflict-messages';
import { FixForAll } from './FixForAll';
import { compareMessages } from './messages';
import { byRelevance, severityOf, typeWords, userFixes } from './problems';
import { UserFix } from './UserFix';
import { UserList } from '../UserList';

function GroupState({ group }: { group: ProblemGroup }) {
  const t = useT(compareMessages);
  if (group.open > 0) {
    return <StatusBadge tone="danger">{t('groupOpen', { count: group.open })}</StatusBadge>;
  }
  const severity = severityOf(group.type);
  const tone: StatusTone =
    severity === 'blocker' ? 'success' : severity === 'warning' ? 'warning' : 'neutral';
  return (
    <StatusBadge tone={tone}>
      {t(severity === 'blocker' ? 'fixed' : `severity_${severity}`)}
    </StatusBadge>
  );
}

/** One problem group: what it is, how many users, the fix for all and, on request, each user. */
function ProblemCard({ group, kind }: { group: ProblemGroup; kind: PanelKind }) {
  const t = useT(compareMessages);
  const tc = useT(conflictMessages);
  const { jobId } = useWizard();
  const [open, setOpen] = useState(false);
  const usersId = useId();
  return (
    <li className="flex flex-col gap-4 border-t border-kumo-hairline px-4 py-4 first:border-0">
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3 className="font-medium text-kumo-strong">
            <CodeText text={tc(typeWords(group.type), { type: group.type })} />
          </h3>
          <span className="text-sm text-kumo-subtle">
            {t('groupUsers', { count: group.count })}
          </span>
        </div>
        <GroupState group={group} />
      </div>
      <FixForAll group={group} kind={kind} />
      {/* A group with one fix has nothing to choose per user. */}
      <div hidden={userFixes(group).length < 2}>
        <Button
          variant="ghost"
          aria-expanded={open}
          aria-controls={usersId}
          className="pointer-coarse:h-11"
          icon={<CaretDownIcon size={14} className={open ? 'rotate-180' : undefined} />}
          onClick={() => setOpen((o) => !o)}
        >
          {t(open ? 'hideUsers' : 'oneByOne')}
        </Button>
      </div>
      <div id={usersId} hidden={!open}>
        {open ? (
          <UserList
            jobId={jobId}
            group={group.type}
            label={t('usersOf')}
            empty={t('noMatch')}
            end={(row) => <UserFix group={group} row={row} />}
          />
        ) : null}
      </div>
    </li>
  );
}

/** Step 3's problem groups, most relevant first (api-migration.md 2.5). */
export function ProblemList({
  groups,
  kind,
}: {
  groups: readonly ProblemGroup[];
  kind: PanelKind;
}) {
  const t = useT(compareMessages);
  return (
    <Section title={t('problemsTitle')}>
      <p className="px-4 pb-3 text-kumo-subtle">{t('problemsIntro')}</p>
      {groups.length === 0 ? (
        <p className="mx-4 mb-4 rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">
          {t('noProblems')}
        </p>
      ) : (
        <ul className="flex flex-col border-t border-kumo-hairline">
          {byRelevance(groups).map((g) => (
            <ProblemCard key={g.type} group={g} kind={kind} />
          ))}
        </ul>
      )}
    </Section>
  );
}
