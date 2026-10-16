import { lazy, Suspense, useState } from 'react';
import { LanguageSwitch } from '../../components/preferences/LanguageSwitch';
import { ThemeToggle } from '../../components/preferences/ThemeToggle';
import { useT } from '../../i18n/locale';
import { sceneAt, type Scene } from './globe-scenes';
import { loginMessages } from './messages';

// Loaded after the form, so the globe never slows down signing in.
const LoginGlobe = lazy(() => import('./LoginGlobe'));

const onBrandButton = 'text-on-brand hover:bg-on-brand/15';

const captions = {
  move: { marzban: 'moveFromMarzban', pasarguard: 'moveFromPasarguard' },
  merge: { marzban: 'mergeFromMarzban', pasarguard: 'mergeFromPasarguard' },
} as const;

function Caption({ scene }: { scene: Scene }) {
  const t = useT(loginMessages);
  return (
    <p className="mt-6 flex min-h-12 items-start gap-2 text-sm text-on-brand/90">
      <span className="shrink-0 rounded bg-on-brand/15 px-1.5 font-medium text-on-brand">
        {t(scene.kind === 'move' ? 'sceneMove' : 'sceneMerge')}
      </span>
      <span>{t(captions[scene.kind][scene.from])}</span>
    </p>
  );
}

/** The brand side of the sign-in page: a spinning globe moving users between panels. */
export function LoginShowcase() {
  const t = useT(loginMessages);
  const [scene, setScene] = useState(0);
  return (
    <section className="relative hidden min-h-dvh flex-col overflow-hidden bg-kumo-brand text-on-brand lg:flex">
      <Suspense fallback={null}>
        <LoginGlobe onScene={setScene} />
      </Suspense>

      <div className="relative flex justify-end gap-1 p-6">
        <LanguageSwitch className={onBrandButton} />
        <ThemeToggle className={onBrandButton} />
      </div>

      <div className="relative flex max-w-lg flex-col px-14 pt-4">
        <p dir="ltr" className="self-start font-mono text-sm text-on-brand/80">
          Marzban ⇄ PasarGuard
        </p>
        <h2 className="mt-3 text-[28px] leading-tight font-bold text-balance">{t('tagline')}</h2>
        <p className="mt-3 text-on-brand/85">{t('taglineDetail')}</p>
        <Caption scene={sceneAt(scene)} />
      </div>
    </section>
  );
}
