import { Button } from '@cloudflare/kumo/components/button';
import { Radio } from '@cloudflare/kumo/components/radio';
import { useNavigate } from 'react-router';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { PanelLabel } from '../../../components/panels/PanelLabel';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import {
  databaseOf,
  engineName,
  panelKind,
  panelRelease,
  type Panel,
} from '../../../lib/api/panels';
import { findingMessages, findingText } from '../../panels/findings';
import { usePanels } from '../../panels/use-panels';
import { connectMessages } from './messages';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

/** Why a panel cannot take users: its first finding, in words. */
function NotReady({ panel }: { panel: Panel }) {
  const t = useT(connectMessages);
  const tf = useT(findingMessages);
  const first = panel.findings[0] as Panel['findings'][number] | undefined;
  const text = first ? findingText(first.code) : undefined;
  return text ? <CodeText text={tf(text.what)} /> : t('destNotReady');
}

function Engine({ panel }: { panel: Panel }) {
  const { engine } = databaseOf(panel);
  return engine ? <bdi dir="ltr">{engineName(engine)}</bdi> : null;
}

/** The destination as radio cards; a panel that is not ready is shown, disabled, with the reason. */
export function DestinationField({
  value,
  onChange,
  disabled,
}: {
  value: number;
  onChange: (id: number) => void;
  disabled?: boolean;
}) {
  const t = useT(connectMessages);
  const navigate = useNavigate();
  const panels = usePanels();
  const load = useFirstLoad(panels.data);
  if (panels.isPending || load.holding) {
    return (
      <div role="status" aria-label={t('destTitle')} className="flex flex-col gap-2">
        <Bone className="h-5 w-36" />
        <Bone className="h-16 w-full" />
        <Bone className="h-16 w-full" />
      </div>
    );
  }
  if (panels.isError) {
    return (
      <ErrorState
        message={t('panelsFailed')}
        error={panels.error}
        onRetry={() => void panels.refetch()}
      />
    );
  }
  const list = panels.data.filter((p) => p.id > 0);
  if (!list.some((p) => p.ready)) {
    return (
      <div className="flex flex-col items-start gap-3">
        <InlineAlert tone="warning">{t('noReadyPanels')}</InlineAlert>
        <Button className="pointer-coarse:h-11" onClick={() => void navigate('/panels')}>
          {t('openPanels')}
        </Button>
      </div>
    );
  }
  return (
    <Reveal on={load.reveal}>
      <Radio.Group
        legend={t('destTitle')}
        description={t('destHint')}
        appearance="card"
        value={String(value)}
        disabled={disabled}
        onValueChange={(v: string) => onChange(Number(v))}
      >
        {list.map((p) => (
          <Radio.Item
            key={p.id}
            value={String(p.id)}
            disabled={!p.ready}
            label={
              <span className="flex flex-col gap-1">
                <PanelLabel
                  kind={panelKind(p)}
                  name={p.name}
                  version={panelRelease(p.revision)}
                  className="font-medium text-kumo-strong"
                />
                <span className="text-sm text-kumo-subtle">
                  {p.ready ? <Engine panel={p} /> : <NotReady panel={p} />}
                </span>
              </span>
            }
          />
        ))}
      </Radio.Group>
    </Reveal>
  );
}
