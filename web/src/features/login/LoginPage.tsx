import { useEffect } from 'react';
import { Wordmark } from '../../components/brand/Wordmark';
import { LanguageSwitch } from '../../components/preferences/LanguageSwitch';
import { ThemeToggle } from '../../components/preferences/ThemeToggle';
import { useT } from '../../i18n/locale';
import { useMedia } from '../../lib/media/use-media';
import { LoginShowcase } from './LoginShowcase';
import { loginMessages } from './messages';
import { SignInForm } from './SignInForm';

export function LoginPage() {
  const t = useT(loginMessages);
  // Under lg only the form shows, and the wall's code is never fetched.
  const wide = useMedia('(min-width: 64rem)');

  useEffect(() => {
    document.title = t('pageTitle');
  }, [t]);

  return (
    <div className="flex min-h-dvh flex-col lg:grid lg:grid-cols-2">
      <main className="flex flex-1 flex-col bg-kumo-base px-5 py-5 sm:px-8 lg:min-h-dvh">
        <header className="flex items-center justify-between gap-3">
          <Wordmark />
          {/* On wide screens these sit on the blue side instead. */}
          <div className="flex items-center gap-1 lg:hidden">
            <LanguageSwitch />
            <ThemeToggle />
          </div>
        </header>

        <div className="mx-auto flex w-full max-w-[22rem] flex-1 flex-col justify-center py-12">
          <SignInForm />
          <p className="mt-6 text-center text-sm text-kumo-subtle">
            {t('lostAccess')}{' '}
            <code
              dir="ltr"
              className="rounded bg-kumo-recessed px-1.5 py-0.5 font-mono text-tiny whitespace-nowrap text-kumo-default"
            >
              karvan reset -password
            </code>
          </p>
        </div>
      </main>
      {wide ? <LoginShowcase /> : null}
    </div>
  );
}
