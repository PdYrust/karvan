import type { ToolbarChoice } from '../../components/data/ListToolbar';
import { PanelMark } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import type { BatchGroup } from '../../lib/api/batches';
import type { PanelKind } from '../../lib/api/panels';
import { batchesMessages } from './messages';

/** One old panel in the filter: its address, its kind's logo, and its runs. */
export interface OldPanelOption {
  origin: string;
  kind: PanelKind;
  runs: number;
}

/**
 * The old panels of a page of groups, first seen first. One address can move
 * onto more than one new panel, so its groups merge into one option and their
 * runs add up.
 */
export function oldPanels(groups: readonly BatchGroup[]): OldPanelOption[] {
  const found = new Map<string, OldPanelOption>();
  for (const g of groups) {
    const next = found.get(g.source.origin);
    if (next) {
      next.runs += g.counts.runs;
    } else {
      found.set(g.source.origin, {
        origin: g.source.origin,
        kind: g.source.kind,
        runs: g.counts.runs,
      });
    }
  }
  return [...found.values()];
}

/** An old panel's logo and address, as every panel address is shown. */
function OldPanelLabel({ option }: { option: OldPanelOption }) {
  return (
    <span className="flex min-w-0 items-center gap-2">
      <PanelMark kind={option.kind} />
      <bdi dir="ltr" className="truncate">
        {option.origin}
      </bdi>
    </span>
  );
}

/** The toolbar's old-panel choice: "all" or one address, with its logo. */
export function useOldPanelChoice(
  groups: readonly BatchGroup[],
  value: string,
  onChange: (value: string) => void,
): ToolbarChoice {
  const t = useT(batchesMessages);
  const options = oldPanels(groups);
  return {
    label: t('oldPanel'),
    value,
    all: 'all',
    options: [
      { value: 'all', label: t('allOldPanels'), text: t('allOldPanels') },
      ...options.map((o) => ({
        value: o.origin,
        label: <OldPanelLabel option={o} />,
        text: o.origin,
        count: o.runs,
      })),
    ],
    onChange,
  };
}
