import { Radio } from '@cloudflare/kumo/components/radio';
import { MoveLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { BatchGroup } from '../../lib/api/batches';
import { siteFormMessages } from './form-messages';
import { rulesMessages } from './messages';

/** One old panel as a choice: the move, its address, the new panel and its runs. */
function GroupLabel({ group }: { group: BatchGroup }) {
  const t = useT(siteFormMessages);
  const tl = useT(rulesMessages);
  const { kind } = group.panel;
  return (
    <span className="flex min-w-0 flex-col gap-1">
      <span className="flex min-w-0 items-center gap-2">
        {kind ? (
          <MoveLabel
            from={group.source.kind}
            to={kind}
            label={tl('move', { from: panelNames[group.source.kind], to: panelNames[kind] })}
          />
        ) : null}
        <bdi dir="ltr" className="min-w-0 font-mono text-[0.9em] break-all text-kumo-strong">
          {group.source.origin}
        </bdi>
      </span>
      <span className="flex flex-wrap gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
        <bdi>{group.panel.name}</bdi>
        <span>{t('groupRuns', { count: group.counts.runs })}</span>
        <span>{t('groupUsers', { count: group.counts.users })}</span>
        {group.counts.active === 0 ? <span>{t('groupNoActive')}</span> : null}
        {group.rules.length > 0 ? (
          <span className="min-w-0">
            {t('groupServed')}
            <bdi>{group.rules.map((r) => r.name).join(', ')}</bdi>
          </span>
        ) : null}
      </span>
    </span>
  );
}

/** The old panels, one choice each: a rule of batches serves every run of the one picked. */
export function OldPanelPicker({
  groups,
  picked,
  error,
  onPick,
}: {
  groups: readonly BatchGroup[];
  picked: number;
  error?: string;
  onPick: (id: number) => void;
}) {
  const t = useT(siteFormMessages);
  return (
    <div className="flex flex-col gap-2">
      {groups.length === 0 ? (
        <div className="flex flex-col gap-1">
          <span className="font-medium text-kumo-strong">{t('batchesTitle')}</span>
          <p className="rounded-lg bg-kumo-tint px-4 py-3 text-sm text-kumo-subtle">
            {t('noBatches')}
          </p>
        </div>
      ) : (
        <Radio.Group
          legend={t('batchesTitle')}
          description={t('batchesHint')}
          appearance="card"
          value={picked ? String(picked) : ''}
          onValueChange={(id: string) => onPick(Number(id))}
          className="w-full"
        >
          {groups.map((g) => (
            <Radio.Item key={g.id} value={String(g.id)} label={<GroupLabel group={g} />} />
          ))}
        </Radio.Group>
      )}
      {error ? (
        <p className="text-sm text-kumo-danger">
          <CodeText text={error} />
        </p>
      ) : null}
    </div>
  );
}
