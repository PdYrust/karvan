import { Checkbox } from '@cloudflare/kumo/components/checkbox';
import { MoveLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { Batch } from '../../lib/api/batches';
import { useFormat } from '../../lib/preferences/region';
import { siteFormMessages } from './form-messages';
import { rulesMessages } from './messages';
import { batchGroups, canPickBatch } from './rule-form';

/**
 * The active batches in groups of one old panel and destination. Picking one
 * leaves only the batches one rule can serve with it (D25).
 */
export function BatchPicker({
  batches,
  picked,
  error,
  onPick,
}: {
  batches: readonly Batch[];
  picked: readonly number[];
  error?: string;
  onPick: (ids: number[]) => void;
}) {
  const t = useT(siteFormMessages);
  const tl = useT(rulesMessages);
  const fmt = useFormat();
  const chosen = batches.filter((b) => picked.includes(b.id));
  return (
    <fieldset className="flex flex-col gap-4">
      <legend className="mb-1 flex flex-col gap-1">
        <span className="font-medium text-kumo-strong">{t('batchesTitle')}</span>
        <span className="text-sm text-kumo-subtle">{t('batchesHint')}</span>
      </legend>
      {batches.length === 0 ? (
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-sm text-kumo-subtle">
          {t('noBatches')}
        </p>
      ) : null}
      {batchGroups(batches).map((g) => (
        <div key={g.key} className="rounded-lg bg-kumo-base ring ring-kumo-hairline">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-kumo-hairline px-4 py-2.5 text-sm">
            {g.panel.kind ? (
              <MoveLabel
                from={g.source.kind}
                to={g.panel.kind}
                label={tl('move', {
                  from: panelNames[g.source.kind],
                  to: panelNames[g.panel.kind],
                })}
              />
            ) : null}
            <bdi dir="ltr" className="min-w-0 font-mono text-[0.9em] break-all text-kumo-strong">
              {g.source.origin}
            </bdi>
            <bdi className="text-kumo-subtle">{g.panel.name}</bdi>
          </div>
          <ul className="flex flex-col divide-y divide-kumo-hairline">
            {g.batches.map((b) => (
              <li key={b.id} className="px-4 py-2.5">
                <Checkbox
                  checked={picked.includes(b.id)}
                  disabled={!canPickBatch(b, chosen)}
                  onCheckedChange={(on) =>
                    onPick(on ? [...picked, b.id] : picked.filter((id) => id !== b.id))
                  }
                  label={
                    <span className="flex min-w-0 flex-col gap-0.5">
                      <bdi className="text-kumo-strong">{b.name}</bdi>
                      <span className="text-sm text-kumo-subtle">
                        {t('batchLine', {
                          count: b.counts.users,
                          date: fmt.date(new Date(b.createdAt)),
                        })}
                      </span>
                    </span>
                  }
                />
              </li>
            ))}
          </ul>
        </div>
      ))}
      {error ? (
        <p className="text-sm text-kumo-danger">
          <CodeText text={error} />
        </p>
      ) : null}
    </fieldset>
  );
}
