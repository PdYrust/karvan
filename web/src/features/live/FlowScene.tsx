import { cn } from '@cloudflare/kumo/utils';
import { useEffect, useMemo, useRef, type ReactNode, type RefObject } from 'react';
import { useWidth } from '../../components/charts/use-width';
import { PanelMark } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import type { PanelKind } from '../../lib/api/panels';
import { chunkDots, curvesDown, curvesFor, pathOf, sinkDots, sinkEnd, type Lane } from './dots';
import type { Counter } from './live';
import { liveMessages } from './messages';
import { SceneTile } from './SceneTile';
import { useDots, type Curves, type Dots } from './use-dots';
import { useStacked } from './use-stacked';

/** The scene's height in pixels; the tiles sit at its middle. */
export const sceneHeight = 144;
/** On phones the dots travel down a lane this tall between the two tiles. */
export const laneHeight = 112;
/** Progress that arrives this soon after the page opens is the stream's history, not a chunk. */
const replayMs = 1500;

const lanes: Readonly<Record<Lane, string>> = {
  forward: 'fill-chart-1',
  back: 'fill-kumo-warning',
  sink: 'fill-current text-kumo-subtle',
};

export interface SceneEnds {
  from: { kind: PanelKind | ''; host: string };
  to: { kind: PanelKind | ''; name: string };
}

/** A panel's tile mark, or nothing when Karvan does not know its kind. */
export function Mark({ kind }: { kind: PanelKind | '' }) {
  return kind ? <PanelMark kind={kind} size={28} /> : null;
}

/** Sends dots in step with each chunk the counter reports. */
function useChunks(counter: Counter | null, skipped: number, dots: Dots) {
  const opened = useRef(0);
  const seen = useRef<{ kind: string; done: number } | null>(null);
  const tally = useRef({ sent: 0, sunk: 0 });
  const kind = counter?.kind ?? '';
  const done = counter?.done ?? 0;
  const total = counter?.total ?? 0;
  useEffect(() => {
    opened.current = performance.now();
  }, []);
  useEffect(() => {
    if (!kind) return;
    const prev = seen.current?.kind === kind ? seen.current.done : null;
    seen.current = { kind, done };
    // The history a stream replays when the page opens is no chunk to show.
    const base = prev ?? (performance.now() - opened.current > replayMs ? 0 : null);
    if (base === null || done <= base) return;
    const n = chunkDots(done - base, total, dots.inFlight());
    const share = kind === 'move' && skipped > 0 ? skipped / (total + skipped) : 0;
    const sunk = sinkDots(tally.current.sent, tally.current.sunk, n, share);
    tally.current = { sent: tally.current.sent + n, sunk: tally.current.sunk + sunk };
    dots.send(n - sunk, kind === 'move' ? 'forward' : 'back');
    dots.send(sunk, 'sink');
  }, [kind, done, total, skipped, dots]);
}

export interface SceneCurves {
  stacked: boolean;
  measure: RefObject<HTMLDivElement | null>;
  width: number;
  curves: Curves | null;
}

/** The scene's curves for its measured width: across from 640 px, standing up under it. */
export function useSceneCurves(sink: boolean): SceneCurves {
  const stacked = useStacked();
  const [measure, width] = useWidth<HTMLDivElement>();
  const curves = useMemo(() => {
    if (width <= 0) return null;
    return stacked ? curvesDown(width, laneHeight, sink) : curvesFor(width, sceneHeight, sink);
  }, [width, sink, stacked]);
  return { stacked, measure, width, curves };
}

/** Lays a scene out: the two tiles at the ends of a row, or above and below on phones. */
export const sceneLayout = (stacked: boolean) =>
  stacked
    ? 'flex flex-col'
    : 'grid grid-cols-[minmax(0,6.5rem)_1fr_minmax(0,6.5rem)] items-center sm:grid-cols-[10rem_1fr_10rem]';

/** The skipped users' count where the sink's branch ends. */
function SinkCount({ stacked, children }: { stacked: boolean; children: ReactNode }) {
  const chip = (
    <span className="rounded-md bg-kumo-tint px-2 py-0.5 text-tiny whitespace-nowrap text-kumo-subtle tabular">
      {children}
    </span>
  );
  if (stacked) {
    return (
      <span
        className="absolute -translate-y-1/2"
        style={{
          top: `${String(sinkEnd.y * 100)}%`,
          insetInlineStart: `calc(${String(sinkEnd.x * 100)}% + 6px)`,
        }}
      >
        {chip}
      </span>
    );
  }
  // As tall as sinkLabel, where the branch ends.
  return (
    <span className="absolute inset-x-0 bottom-0 flex h-7 items-start justify-center">{chip}</span>
  );
}

/** The middle of a scene: the dashed arc, the sink's branch and the dots, sized in pixels. */
export function ArcLayer({
  scene,
  circle,
  children,
}: {
  scene: SceneCurves;
  circle: readonly ((node: SVGCircleElement | null) => void)[];
  children?: ReactNode;
}) {
  const { measure, width, curves, stacked } = scene;
  const height = stacked ? laneHeight : sceneHeight;
  return (
    <div ref={measure} style={{ height }} className={cn('relative', !stacked && '-mx-1')}>
      {curves ? (
        <svg
          aria-hidden="true"
          width={width}
          height={height}
          className="absolute inset-0 overflow-visible rtl:-scale-x-100"
        >
          <path
            d={pathOf(curves.arc)}
            strokeDasharray="3 5"
            strokeWidth={1.5}
            strokeOpacity={0.5}
            className="fill-none stroke-current text-kumo-subtle"
          />
          {curves.sink ? (
            <path
              d={pathOf(curves.sink)}
              strokeDasharray="3 5"
              strokeWidth={1.5}
              strokeOpacity={0.35}
              className="fill-none stroke-current text-kumo-subtle"
            />
          ) : null}
          {circle.map((ref, i) => (
            <circle key={i} ref={ref} r={4.5} cx={0} cy={0} opacity={0} />
          ))}
        </svg>
      ) : null}
      {children}
    </div>
  );
}

/**
 * Move and undo (4.6): the two panels as tiles, a dashed arc between them
 * in the page direction, dots in step with each chunk, and the skipped
 * users' sink. The drawing is decoration; every number is text.
 */
export function FlowScene({
  mode,
  counter,
  skipped,
  ends,
}: {
  mode: 'move' | 'undo';
  counter: Counter | null;
  skipped: number;
  ends: SceneEnds;
}) {
  const t = useT(liveMessages);
  const sink = mode === 'move' && skipped > 0;
  const scene = useSceneCurves(sink);
  const { stacked } = scene;
  const [box, dots] = useDots(scene.curves, lanes);
  useChunks(counter, skipped, dots);
  const done = counter?.done ?? 0;
  const total = counter?.total ?? 0;
  return (
    <div ref={box} className={sceneLayout(stacked)}>
      <SceneTile
        row={stacked}
        mark={<Mark kind={ends.from.kind} />}
        name={ends.from.kind ? panelNames[ends.from.kind] : ''}
        detail={ends.from.host}
        value={mode === 'move' && counter ? t('toGo', { count: Math.max(0, total - done) }) : null}
      />
      <ArcLayer scene={scene} circle={dots.circle}>
        {sink ? (
          <SinkCount stacked={stacked}>{t('skippedCount', { count: skipped })}</SinkCount>
        ) : null}
      </ArcLayer>
      <SceneTile
        row={stacked}
        mark={<Mark kind={ends.to.kind} />}
        name={ends.to.kind ? panelNames[ends.to.kind] : ends.to.name}
        detail={ends.to.name}
        value={counter ? t(mode === 'move' ? 'movedCount' : 'removedCount', { count: done }) : null}
      />
    </div>
  );
}
