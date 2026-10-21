import { Button } from '@cloudflare/kumo/components/button';
import { HourglassIcon, QuestionIcon, WarningCircleIcon } from '@phosphor-icons/react';
import { useLocation, useNavigate, useParams } from 'react-router';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import { ApiError } from '../../../lib/api/client';
import { analyticsMessages } from '../messages';
import { databaseRange, rangeSpan, useRange } from '../range';
import { LoadFailed, Notice, TabSkeleton } from '../states';
import { usePanelStats } from '../use-analytics';
import { EngineCard } from './EngineCard';
import { HistoryCards } from './HistoryCards';
import { AdminsCard, ClientsCard, NodesCard, TablesCard } from './ListCards';
import { findingMessages, findingText } from '../../panels/findings';
import { PanelTiles } from './PanelTiles';
import { StatusCard, UsersTrendCard } from './UsersCards';

const iconProps = { size: 40, weight: 'light', 'aria-hidden': true } as const;

function UnknownPanel() {
  const t = useT(analyticsMessages);
  const { search } = useLocation();
  const navigate = useNavigate();
  return (
    <Notice
      icon={<QuestionIcon {...iconProps} />}
      action={
        <Button
          variant="secondary"
          className="pointer-coarse:h-11"
          onClick={() => void navigate(`/analytics/server${search}`)}
        >
          {t('backToServer')}
        </Button>
      }
    >
      {t('unknownPanel')}
    </Notice>
  );
}

/** The finding that keeps Karvan out of the database, with how to fix it. */
function Problem({ code }: { code: string }) {
  const t = useT(analyticsMessages);
  const tf = useT(findingMessages);
  const text = findingText(code);
  return (
    <Notice icon={<WarningCircleIcon {...iconProps} />}>
      {text ? <CodeText text={tf(text.what)} /> : t('problemOther')}
      {text ? (
        <span className="mt-1 block text-sm text-kumo-subtle">
          <CodeText text={tf(text.fix)} />
        </span>
      ) : null}
    </Notice>
  );
}

/** One panel database: tiles, users, traffic, the engine, tables, admins, nodes and apps. */
export function PanelTab() {
  const t = useT(analyticsMessages);
  const id = Number(useParams().id);
  const [range] = useRange();
  const valid = Number.isInteger(id) && id > 0;
  const q = usePanelStats(valid ? id : 0, range);

  if (!valid || (q.error instanceof ApiError && q.error.status === 404 && !q.data)) {
    return <UnknownPanel />;
  }
  if (!q.data) {
    return q.isError ? (
      <LoadFailed error={q.error} onRetry={() => void q.refetch()} />
    ) : (
      <TabSkeleton label={t('title')} />
    );
  }
  const { panel, detail, detailProblem } = q.data;
  if (panel.problem) return <Problem code={panel.problem.code} />;
  if (panel.pending || !panel.data) {
    return <Notice icon={<HourglassIcon {...iconProps} />}>{t('pending')}</Notice>;
  }
  const end = Date.parse(panel.at) || (panel.points.at(-1)?.t ?? 0) * 1000;
  const step = q.data.step * 1000;
  const domain: [number, number] = [end - rangeSpan[databaseRange(range)], end];
  const missing =
    detailProblem === undefined
      ? undefined
      : t(detailProblem === 'pending' ? 'pending' : 'detailUnreachable');
  return (
    <>
      <PanelTiles panel={panel} data={panel.data} step={step} domain={domain} />
      {/* Pairs of about the same height share a row; the lists below vary, so they stack in two columns. */}
      <div className="grid gap-4 lg:grid-cols-2">
        <StatusCard data={panel.data} />
        <UsersTrendCard panel={panel} data={panel.data} step={step} domain={domain} range={range} />
        <HistoryCards detail={detail} missing={missing} end={end} range={range} />
      </div>
      <div className="grid items-start gap-4 lg:grid-cols-2">
        <div className="flex flex-col gap-4">
          <EngineCard panel={panel} step={step} domain={domain} range={range} />
          <AdminsCard data={panel.data} />
        </div>
        <div className="flex flex-col gap-4">
          <TablesCard detail={detail} missing={missing} />
          <NodesCard data={panel.data} />
          <ClientsCard detail={detail} missing={missing} kind={panel.kind} range={range} />
        </div>
      </div>
    </>
  );
}
