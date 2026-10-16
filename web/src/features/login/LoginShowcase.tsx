import { lazy, Suspense } from 'react';
import { LanguageSwitch } from '../../components/preferences/LanguageSwitch';
import { ThemeToggle } from '../../components/preferences/ThemeToggle';
import { useT } from '../../i18n/locale';
import { loginMessages } from './messages';

// Loaded after the form, so the globe never slows down signing in.
const LoginGlobe = lazy(() => import('./LoginGlobe'));

const onBrandButton = 'text-on-brand hover:bg-on-brand/15';

/** The brand side of the sign-in page: a spinning globe moving users between panels. */
export function LoginShowcase() {
  const t = useT(loginMessages);
  return (
    <section className="relative hidden min-h-dvh flex-col overflow-hidden bg-kumo-brand text-on-brand lg:flex">
      <Suspense fallback={null}>
        <LoginGlobe />
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
      </div>
    </section>
  );
}
