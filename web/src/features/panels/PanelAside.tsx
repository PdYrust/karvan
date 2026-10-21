import { StatusBadge } from '../../components/feedback/StatusBadge';
import { AsideBlock } from '../../components/layout/AsideBlock';
import { FactList, type Fact } from '../../components/layout/FactList';
import { CodeText } from '../../components/text/CodeText';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import { databaseOf, engineName, type Panel, type PanelControl } from '../../lib/api/panels';
import { controlMessages, controlText } from './control';
import { panelsMessages } from './messages';

/** Can Karvan stop and start the panel; if not, why, how to allow it, and what that means. */
function ControlBlock({ control }: { control: PanelControl }) {
  const t = useT(panelsMessages);
  const tc = useT(controlMessages);
  const text = controlText(control);
  return (
    <AsideBlock
      title={t('controlTitle')}
      badge={
        <StatusBadge tone={control.available ? 'success' : 'neutral'}>
          {tc(control.available ? 'available' : 'unavailable')}
        </StatusBadge>
      }
    >
      <p className="text-sm text-kumo-default">
        <CodeText text={tc(text.hint)} />
      </p>
      {text.fix ? (
        <p className="text-sm text-kumo-subtle">
          <CodeText text={tc(text.fix)} />
        </p>
      ) : null}
      {control.available ? null : <p className="text-sm text-kumo-subtle">{tc('undoOnly')}</p>}
    </AsideBlock>
  );
}

/** The side column of a panel's page: panel control, then the short facts. */
export function PanelAside({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  const { engine } = databaseOf(panel);
  const facts: (Fact | false)[] = [
    !!panel.revision && {
      label: t('factRevision'),
      value: <Machine>{panel.revision}</Machine>,
    },
    { label: t('factDatabase'), value: engine ? engineName(engine) : t('notRead') },
  ];
  return (
    <div className="flex flex-col">
      {panel.control ? <ControlBlock control={panel.control} /> : null}
      <AsideBlock title={t('factsTitle')}>
        <FactList facts={facts.filter((f): f is Fact => Boolean(f))} />
      </AsideBlock>
    </div>
  );
}
