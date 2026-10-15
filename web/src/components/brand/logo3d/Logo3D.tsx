import { cn } from '@cloudflare/kumo/utils';
import { useEffect, useRef, useState } from 'react';
import { Mark } from '../Mark';

/** The 3D star is skipped where it would cost more than it gives. */
function canRender3D(): boolean {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return false;
  const nav = navigator as Navigator & {
    deviceMemory?: number;
    connection?: { saveData?: boolean };
  };
  if ((nav.deviceMemory ?? 8) <= 2 || nav.connection?.saveData) return false;
  return Boolean(document.createElement('canvas').getContext('webgl2'));
}

/**
 * The login stage's star. The flat mark shows at once; the 3D one loads
 * after the page is usable and fades in over it.
 */
export function Logo3D({ className }: { className?: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [shown, setShown] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !canRender3D()) return;
    let cancelled = false;
    let cleanup: () => void = () => undefined;

    void Promise.all([import('./scene'), import('./palette')])
      .then(([{ createLogoScene }, { readPalette }]) => {
        if (cancelled) return;
        const scene = createLogoScene(canvas, readPalette());
        const resize = () => scene.resize(canvas.clientWidth, canvas.clientHeight);
        const sizeObserver = new ResizeObserver(resize);
        sizeObserver.observe(canvas);
        resize();
        const themeObserver = new MutationObserver(() => scene.setPalette(readPalette()));
        themeObserver.observe(document.documentElement, { attributeFilter: ['data-mode'] });
        const onMove = (e: PointerEvent) =>
          scene.setPointer(
            (e.clientX / window.innerWidth) * 2 - 1,
            (e.clientY / window.innerHeight) * 2 - 1,
          );
        const onLeave = () => scene.setPointer(null, null);
        window.addEventListener('pointermove', onMove, { passive: true });
        document.addEventListener('pointerleave', onLeave);
        scene.start(
          () => setShown(true),
          () => setFailed(true),
        );
        cleanup = () => {
          window.removeEventListener('pointermove', onMove);
          document.removeEventListener('pointerleave', onLeave);
          sizeObserver.disconnect();
          themeObserver.disconnect();
          scene.dispose();
        };
      })
      .catch(() => setFailed(true));

    return () => {
      cancelled = true;
      cleanup();
    };
  }, []);

  const live = shown && !failed;
  return (
    <div className={cn('relative', className)}>
      <Mark
        size="62%"
        className={cn(
          'absolute inset-0 m-auto transition-opacity duration-700',
          live ? 'opacity-0' : 'opacity-100',
        )}
      />
      {failed ? null : (
        <canvas
          ref={canvasRef}
          aria-hidden="true"
          className={cn(
            'absolute inset-0 size-full transition-opacity duration-700',
            live ? 'opacity-100' : 'opacity-0',
          )}
        />
      )}
    </div>
  );
}
