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
  const gl = document.createElement('canvas').getContext('webgl2');
  if (!gl) return false;
  // A software rasterizer draws on the CPU and would slow the whole page.
  const info = gl.getExtension('WEBGL_debug_renderer_info');
  const renderer = info ? String(gl.getParameter(info.UNMASKED_RENDERER_WEBGL)) : '';
  gl.getExtension('WEBGL_lose_context')?.loseContext();
  return !/swiftshader|llvmpipe|softpipe|software/i.test(renderer);
}

/** Waits until the page has loaded and the browser has a quiet moment. */
function whenIdle(): Promise<void> {
  return new Promise((resolve) => {
    // Safari has no requestIdleCallback.
    const ric = window.requestIdleCallback as typeof window.requestIdleCallback | undefined;
    const idle = () =>
      ric
        ? window.requestIdleCallback(() => resolve(), { timeout: 2000 })
        : setTimeout(resolve, 200);
    if (document.readyState === 'complete') idle();
    else window.addEventListener('load', idle, { once: true });
  });
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

    void whenIdle()
      .then(() => Promise.all([import('./scene'), import('./palette')]))
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
