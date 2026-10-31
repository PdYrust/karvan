import { ArchiveIcon } from '@phosphor-icons/react';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import type { Lane } from './dots';
import { ArcLayer, Mark, sceneLayout, useSceneCurves, type SceneEnds } from './FlowScene';
import { liveMessages } from './messages';
import { SceneTile } from './SceneTile';
import { useDots } from './use-dots';

const lanes: Readonly<Record<Lane, string>> = {
  forward: 'fill-chart-4',
  back: 'fill-chart-4',
  sink: 'fill-chart-4',
};

/**
 * A backup going back: the backup's tile, the panel offline at the
 * other end, and dots flowing into it while the restore runs. A restore has
 * no chunks, so the dots say only that it is under way; the timer is text.
 * The panel reads Online again once it started.
 */
export function RewindScene({
  ends,
  backupId,
  online,
}: {
  ends: SceneEnds;
  backupId: string;
  online: boolean;
}) {
  const t = useT(liveMessages);
  const scene = useSceneCurves(false);
  const [box, dots] = useDots(scene.curves, lanes, 'forward');
  return (
    <div ref={box} className={sceneLayout(scene.stacked)}>
      <SceneTile
        row={scene.stacked}
        mark={<ArchiveIcon size={28} aria-hidden="true" className="text-kumo-default" />}
        name={t('backup')}
        detail={backupId}
      />
      <ArcLayer scene={scene} circle={dots.circle} />
      <SceneTile
        row={scene.stacked}
        dim={!online}
        mark={<Mark kind={ends.to.kind} />}
        name={ends.to.kind ? panelNames[ends.to.kind] : ends.to.name}
        detail={ends.to.name}
        value={t(online ? 'online' : 'offline')}
      />
    </div>
  );
}
