import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { FactList } from '../../../components/layout/FactList';
import { Section } from '../../../components/layout/Section';
import { PanelLabel } from '../../../components/panels/PanelLabel';
import { useT } from '../../../i18n/locale';
import { getSummary, migrationKeys, type JobSummary } from '../../../lib/api/migrations';
import { errorOf } from '../../../lib/errors/errors';
import { useFormat } from '../../../lib/preferences/region';
import { usePanels } from '../../panels/use-panels';
import { wizardMessages } from '../messages';
import { StepActions } from '../StepActions';
import { summaryMessages } from './messages';
import { SummaryCards } from './SummaryCards';
import { TargetList } from './TargetList';

function Panels({ summary }: { summary: JobSummary }) {
  const t = useT(summaryMessages);
  const fmt = useFormat();
  const dest = usePanels().data?.find((p) => p.id === summary.panelId);
  const byId = new Map((summary.targets ?? []).map((g) => [g.id, g.name]));
  const going = (summary.userTargets ?? []).map((id) => byId.get(id) ?? id);
  const source = summary.sourceKind || undefined;
  const destKind = summary.destKind || undefined;
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Section title={t('oldPanel')}>
        <div className="px-4 pb-2">
          <FactList
            facts={[
              { label: t('panel'), value: <PanelLabel kind={source} /> },
              { label: t('address'), value: <bdi dir="ltr">{summary.origin}</bdi> },
              { label: t('admin'), value: <bdi>{summary.admin || summary.newAdmin.username}</bdi> },
              { label: t('users'), value: fmt.number(summary.users) },
              { label: t('usedTraffic'), value: fmt.bytes(summary.usedTraffic) },
            ]}
          />
        </div>
      </Section>
      <Section title={t('destination')}>
        <div className="px-4 pb-2">
          <FactList
            facts={[
              { label: t('panel'), value: <PanelLabel kind={destKind} name={dest?.name} /> },
              {
                label: t('usersGo'),
                value: going.length ? <bdi>{going.join(', ')}</bdi> : t('nowhere'),
              },
            ]}
          />
        </div>
      </Section>
    </div>
  );
}

/** Step 2: what Karvan read from the old panel, and where its users can go. */
export function SummaryStep() {
  const { id = '' } = useParams();
  const t = useT(summaryMessages);
  const tw = useT(wizardMessages);
  const summary = useQuery({ queryKey: migrationKeys.summary(id), queryFn: () => getSummary(id) });
  if (summary.isPending) {
    return (
      <div role="status" aria-label={t('loading')} className="flex flex-col gap-4">
        <div className="grid gap-4 lg:grid-cols-2">
          <Bone className="h-56 w-full" />
          <Bone className="h-56 w-full" />
        </div>
        <Bone className="h-48 w-full" />
      </div>
    );
  }
  if (summary.isError) {
    return errorOf(summary.error).code === 'not_checked' ? (
      <InlineAlert tone="warning">{tw('notChecked')}</InlineAlert>
    ) : (
      <ErrorState
        message={t('failed')}
        error={summary.error}
        onRetry={() => void summary.refetch()}
      />
    );
  }
  return (
    <div className="flex flex-col gap-6">
      <Panels summary={summary.data} />
      <SummaryCards summary={summary.data} />
      <TargetList summary={summary.data} />
      <StepActions jobId={id} next="compare" />
    </div>
  );
}
