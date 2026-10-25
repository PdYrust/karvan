import { ArchiveIcon } from '@phosphor-icons/react';
import { useMemo } from 'react';
import { useWidth } from '../../components/charts/use-width';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import { curvesFor, type Lane } from './dots';
import { ArcLayer, Mark, sceneHeight, type SceneEnds } from './FlowScene';
import { liveMessages } from './messages';
import { SceneTile } from './SceneTile';
import { useDots } from './use-dots';

const lanes: Readonly<Record<Lane, string>> = {
  forward: 'fill-chart-4',
  back: 'fill-chart-4',
  sink: 'fill-chart-4',
};

/**
 * A backup going back (4.6): the backup's tile, the panel offline at the
 * other end, and dots flowing into it while the restore runs. A restore has
 * no batches, so the dots say only that it is under way; the timer is text.
 */
export function RewindScene({ ends, backupId }: { ends: SceneEnds; backupId: string }) {
  const t = useT(liveMessages);
  const [measure, width] = useWidth<HTMLDivElement>();
  const curves = useMemo(() => (width > 0 ? curvesFor(width, sceneHeight, false) : null), [width]);
  const [box, dots] = useDots(curves, lanes, 'forward');
  return (
    <div
      ref={box}
      className="grid grid-cols-[minmax(0,6.5rem)_1fr_minmax(0,6.5rem)] items-center sm:grid-cols-[10rem_1fr_10rem]"
    >
      <SceneTile
        mark={<ArchiveIcon size={28} aria-hidden="true" className="text-kumo-default" />}
        name={t('backup')}
        detail={backupId}
      />
      <ArcLayer measure={measure} width={width} curves={curves} circle={dots.circle} />
      <SceneTile
        dim
        mark={<Mark kind={ends.to.kind} />}
        name={ends.to.kind ? panelNames[ends.to.kind] : ends.to.name}
        detail={ends.to.name}
        value={t('offline')}
      />
    </div>
  );
}
