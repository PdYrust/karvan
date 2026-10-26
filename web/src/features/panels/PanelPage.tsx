import { Button } from '@cloudflare/kumo/components/button';
import { ArrowsClockwiseIcon, ChartLineIcon } from '@phosphor-icons/react';
import { Link, useNavigate, useParams } from 'react-router';
import { ErrorState } from '../../components/feedback/ErrorState';
import { PageSkeleton } from '../../components/feedback/PageSkeleton';
import { StatusBadge } from '../../components/feedback/StatusBadge';
import { PageHeader } from '../../components/layout/PageHeader';
import { Section } from '../../components/layout/Section';
import { usePageTitle } from '../../components/layout/use-page-title';
import { PanelMark } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import { panelKind, type Panel } from '../../lib/api/panels';
import { useCurrentPanel } from './current-panel';
import { FindingList } from './FindingList';
import { panelsMessages } from './messages';
import { PanelAside } from './PanelAside';
import { CurrentTag, PanelVersion } from './PanelCells';
import { PanelEnv } from './PanelEnv';
import { usePanels, useRefreshPanels } from './use-panels';
import { Reveal } from '../../components/feedback/Reveal';
import { useFirstLoad } from '../../lib/motion/use-first-load';

/** The breadcrumb above the title, back to the list. */
function Crumb() {
  const t = useT(panelsMessages);
  return (
    <Link to="/panels" className="hover:text-kumo-default hover:underline">
      {t('title')}
    </Link>
  );
}

function Actions({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  const navigate = useNavigate();
  const { panel: current, select } = useCurrentPanel();
  const { refresh, refreshing } = useRefreshPanels();
  return (
    <>
      <Button
        icon={<ArrowsClockwiseIcon size={16} />}
        loading={refreshing}
        onClick={refresh}
        className="pointer-coarse:h-11"
      >
        {t('check')}
      </Button>
      {current?.id === panel.id ? null : (
        <Button onClick={() => select(panel.id)} className="pointer-coarse:h-11">
          {t('workOn')}
        </Button>
      )}
      <Button
        icon={<ChartLineIcon size={16} />}
        onClick={() => void navigate(`/analytics/panels/${String(panel.id)}`)}
        className="pointer-coarse:h-11"
      >
        {t('analytics')}
      </Button>
    </>
  );
}

function Detail({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  const { panel: current } = useCurrentPanel();
  const kind = panelKind(panel);
  const problems = panel.findings.length;
  return (
    <div className="flex flex-col">
      <PageHeader
        eyebrow={<Crumb />}
        title={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            {kind ? <PanelMark kind={kind} size={24} /> : null}
            <bdi className="min-w-0 break-all">{panel.name}</bdi>
            <StatusBadge tone={panel.ready ? 'success' : 'warning'}>
              {panel.ready ? t('ready') : t('problems', { count: problems })}
            </StatusBadge>
            {current?.id === panel.id ? <CurrentTag /> : null}
          </span>
        }
        description={<PanelVersion panel={panel} />}
        actions={<Actions panel={panel} />}
      />
      <div className="grid max-w-[1400px] lg:grid-cols-[minmax(0,1fr)_22rem]">
        <div className="flex min-w-0 flex-col gap-6 p-4 sm:p-8">
          {problems > 0 ? (
            <Section title={t('problemsTitle')}>
              <div className="px-4 pt-1 pb-5">
                <FindingList findings={panel.findings} />
              </div>
            </Section>
          ) : null}
          <PanelEnv panel={panel} />
        </div>
        <aside
          aria-label={t('factsTitle')}
          className="border-t border-kumo-hairline p-4 sm:p-8 lg:border-s lg:border-t-0"
        >
          <PanelAside panel={panel} />
        </aside>
      </div>
    </div>
  );
}

/** One panel: what is wrong and how to fix it, what Karvan read, and whether it can stop the panel. */
export function PanelPage() {
  const t = useT(panelsMessages);
  const navigate = useNavigate();
  const id = Number(useParams().id);
  const panels = usePanels();
  const load = useFirstLoad(panels.data);
  const panel = load.data?.find((p) => p.id === id);
  usePageTitle(panel?.name ?? t('title'));

  if (panel) {
    return (
      <Reveal on={load.reveal}>
        <Detail panel={panel} />
      </Reveal>
    );
  }
  if (!load.data && !panels.isError) return <PageSkeleton label={t('title')} />;
  if (!load.data) {
    return (
      <div className="flex flex-col">
        <PageHeader title={t('title')} />
        <div className="p-4 sm:p-8">
          <ErrorState
            message={t('loadFailed')}
            error={panels.error}
            onRetry={() => void panels.refetch()}
          />
        </div>
      </div>
    );
  }
  return (
    <PageHeader
      eyebrow={<Crumb />}
      title={t('unknownTitle')}
      description={t('unknownBody')}
      actions={
        <Button onClick={() => void navigate('/panels')} className="pointer-coarse:h-11">
          {t('allPanels')}
        </Button>
      }
    />
  );
}
