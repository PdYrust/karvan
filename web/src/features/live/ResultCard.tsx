import { Button } from '@cloudflare/kumo/components/button';
import { LinkIcon } from '@phosphor-icons/react';
import { useNavigate } from 'react-router';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { Bone } from '../../components/feedback/PageSkeleton';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { newSitePath } from '../../lib/api/links';
import type { JobRow } from '../../lib/api/migrations';
import type { Report } from '../../lib/api/report';
import { useFormat } from '../../lib/preferences/region';
import { jobStates } from '../migrations/job-state';
import { stepPath } from '../wizard/steps';
import { Files } from './Files';
import { mayRollBack, shownCounts, type EndState } from './live';
import { resultHeads, useMeaning, useReload } from './meaning';
import { liveMessages } from './messages';
import { RollBackButton } from './RollBackButton';

function ReloadLine({ report }: { report: Report }) {
  const t = useT(liveMessages);
  const r = useReload(report);
  if (!r) return null;
  return (
    <div className="flex flex-col gap-1">
      <h3 className="font-medium text-kumo-strong">{t('reloadTitle')}</h3>
      <p className="text-kumo-default">
        <CodeText text={r.text} />
      </p>
    </div>
  );
}

/**
 * When a job ends (4.6): its state, what that means, the counts, the
 * reload, the files, and what to do next.
 */
export function ResultCard({
  jobId,
  state,
  row,
  report,
  reportLoading,
}: {
  jobId: string;
  state: EndState;
  row: JobRow;
  report: Report | undefined;
  reportLoading: boolean;
}) {
  const t = useT(liveMessages);
  const fmt = useFormat();
  const navigate = useNavigate();
  const meaning = useMeaning(state, report);
  const look = jobStates[state];
  const counts = report?.counts ?? row.counts;
  const moved = (counts?.moved ?? 0) - (counts?.undone ?? 0);
  const backup = report?.safeMode === true && report.rollback === 'backup';
  // A rollback took the new admin back with the users, so its password is no use then.
  const password =
    report?.admin.password === 'generated' && (state === 'done' || state === 'failed')
      ? report.admin.username
      : null;
  return (
    <section
      aria-labelledby="live-result"
      className="flex flex-col gap-5 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-6"
    >
      <div className="flex flex-col gap-2">
        <h2 id="live-result" className="text-heading font-semibold text-kumo-strong">
          <StatusLabel tone={state === 'checked' ? 'warning' : look.tone}>
            {t(resultHeads[state])}
          </StatusLabel>
        </h2>
        {meaning.map((m) => (
          <p key={m} className="max-w-[75ch] text-kumo-default">
            {m}
          </p>
        ))}
      </div>
      {counts && state !== 'checked' && state !== 'interrupted' ? (
        <dl aria-label={t('counts')} className="flex flex-wrap gap-x-8 gap-y-3">
          {shownCounts(counts).map(([k, n]) => (
            <div key={k} className="flex flex-col">
              <dt className="text-sm text-kumo-subtle">{t(`count_${k}`)}</dt>
              <dd className="text-title font-semibold text-kumo-strong tabular">{fmt.number(n)}</dd>
            </div>
          ))}
        </dl>
      ) : null}
      {reportLoading ? <Bone className="h-16 w-full" /> : null}
      {report && state === 'done' ? <ReloadLine report={report} /> : null}
      {report ? (
        <Files jobId={jobId} links={state === 'done' && moved > 0} password={password} />
      ) : null}
      <div className="flex flex-wrap items-center gap-2 border-t border-kumo-hairline pt-4">
        {state === 'checked' ? (
          <Button
            variant="primary"
            className="pointer-coarse:h-9"
            onClick={() => void navigate(stepPath('start', jobId))}
          >
            {t('reviewChanges')}
          </Button>
        ) : null}
        {mayRollBack(state, counts, report) ? (
          <RollBackButton jobId={jobId} backup={backup} />
        ) : null}
        {state === 'done' && moved > 0 && row.source.kind ? (
          <Button
            variant="primary"
            icon={<LinkIcon size={16} />}
            className="pointer-coarse:h-9"
            title={t('oldLinksHint')}
            onClick={() =>
              void navigate(
                newSitePath({
                  panelId: row.panel.id,
                  sourceKind: row.source.kind,
                  sourceUrl: row.source.origin,
                }),
              )
            }
          >
            {t('oldLinks')}
          </Button>
        ) : null}
      </div>
    </section>
  );
}
