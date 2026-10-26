import { lazy, Suspense } from 'react';
import { LanguageSwitch } from '../../components/preferences/LanguageSwitch';
import { ThemeToggle } from '../../components/preferences/ThemeToggle';
import { useT } from '../../i18n/locale';
import { loginMessages } from './messages';

// Loaded after the form, so the wall never slows down signing in.
const LoginWall = lazy(() => import('./LoginWall'));

const onBrandButton = 'text-on-brand hover:bg-on-brand/15';

/**
 * The brand side of the sign-in page, the firewall as a wall of hexagonal
 * columns. On small screens it is a band under the form, with nothing on it.
 */
export function LoginShowcase() {
  const t = useT(loginMessages);
  return (
    <section className="relative flex h-[22dvh] max-h-44 min-h-28 flex-col overflow-hidden bg-kumo-brand text-on-brand lg:h-auto lg:max-h-none lg:min-h-dvh">
      <Suspense fallback={null}>
        <LoginWall />
      </Suspense>

      <div className="relative hidden justify-end gap-1 p-6 lg:flex">
        <LanguageSwitch className={onBrandButton} />
        <ThemeToggle className={onBrandButton} />
      </div>

      <div className="relative hidden max-w-lg flex-col px-14 pt-4 lg:flex">
        <p dir="ltr" className="self-start font-mono text-sm text-on-brand/80">
          Marzban ⇄ PasarGuard
        </p>
        <h2 className="mt-3 text-[28px] leading-tight font-bold text-balance">{t('tagline')}</h2>
      </div>
    </section>
  );
}
