import { Button } from '@cloudflare/kumo/components/button';
import { ArrowsClockwiseIcon, CheckCircleIcon } from '@phosphor-icons/react';
import { useId } from 'react';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { useT } from '../../../i18n/locale';
import type { Changes } from '../../../lib/api/preflight';
import { useErrorText } from '../../../lib/errors/errors';
import { ReadProgress } from '../connect/ReadProgress';
import { ChangesDiff } from './ChangesDiff';
import { startMessages } from './messages';
import type { useCheck } from './use-check';
import { Checkbox } from '@cloudflare/kumo/components/checkbox';
type Check = ReturnType<typeof useCheck>;
/** The admin and the job's own problems, when the new read changed them. */
function JobChanges({ c }: { c: Changes }) {
  const t = useT(startMessages);
  const lines: string[] = [];
  const a = c.admin;
  if (a?.fields.includes('username')) {
    lines.push(t('adminChanged', { before: a.before.username, after: a.after.username }));
  }
  if (a?.fields.includes('merged'))
    lines.push(t(a.after.merged ? 'adminMergeOn' : 'adminMergeOff'));
  if (a?.fields.includes('roleId')) lines.push(t('adminRole'));
  if (c.conflicts) lines.push(t('conflictsChanged'));
  if (lines.length === 0) return null;
  return (
    <ul className="flex list-disc flex-col gap-1 ps-5">
      {lines.map((l) => (
        <li key={l}>{l}</li>
      ))}
    </ul>
  );
}
function Result({ check }: { check: Check }) {
  const t = useT(startMessages);
  const changes = check.changes;
  if (!changes) return null;
  if (changes.isError) {
    return (
      <ErrorState
        message={t('checkFailed')}
        error={changes.error}
        onRetry={() => void changes.refetch()}
      />
    );
  }
  const c = changes.data;
  if (!c) return <Bone className="h-16 w-full" />;
  if (!c.changed) {
    return (
      <p role="status" className="flex items-center gap-2 text-kumo-default">
        <CheckCircleIcon size={18} weight="bold" aria-hidden="true" className="text-kumo-success" />
        {t('nothingChanged')}
      </p>
    );
  }
  return (
    <div className="flex flex-col gap-4">
      <InlineAlert tone="warning">{t('somethingChanged')}</InlineAlert>
      <JobChanges c={c} />
      <ChangesDiff changes={c} offset={check.offset} onOffset={check.setOffset} />
      <Checkbox label={t('accept')} checked={check.accepted} onCheckedChange={check.setAccepted} />
    </div>
  );
}
/** Reads both panels again and says what changed since the operator reviewed. */
export function CheckAgain({ check }: { check: Check }) {
  const t = useT(startMessages);
  const errorText = useErrorText();
  const titleId = useId();
  if (check.status === 'reading' || check.status === 'building') {
    return <ReadProgress stream={check.stream} onStop={check.stop} stopping={check.stopping} />;
  }
  return (
    <section
      aria-labelledby={titleId}
      className="flex flex-col gap-4 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-5"
    >
      <div className="flex flex-col gap-1">
        <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
          {t('checkTitle')}
        </h2>
        <p className="max-w-[720px] text-kumo-subtle">{t('checkBody')}</p>
      </div>
      <div>
        <Button
          icon={<ArrowsClockwiseIcon size={16} />}
          loading={check.compare.isPending}
          disabled={!check.ready}
          className="pointer-coarse:h-9"
          onClick={check.run}
        >
          {t('checkButton')}
        </Button>
      </div>
      {check.compare.isError ? (
        <InlineAlert>{`${t('checkFailed')} ${errorText(check.compare.error)}`}</InlineAlert>
      ) : null}
      {check.failure ? (
        <InlineAlert>{`${t('checkFailed')} ${errorText(check.failure)}`}</InlineAlert>
      ) : null}
      {check.status === 'built' ? <Result check={check} /> : null}
    </section>
  );
}
