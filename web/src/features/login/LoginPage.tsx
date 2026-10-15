import { useEffect } from 'react';
import { Wordmark } from '../../components/brand/Wordmark';
import { LanguageSwitch } from '../../components/preferences/LanguageSwitch';
import { ThemeToggle } from '../../components/preferences/ThemeToggle';
import { useT } from '../../i18n/locale';
import { LoginStage } from './LoginStage';
import { loginMessages } from './messages';
import { SignInForm } from './SignInForm';

export function LoginPage() {
  const t = useT(loginMessages);

  useEffect(() => {
    document.title = t('pageTitle');
  }, [t]);

  return (
    <div className="grid min-h-dvh grid-rows-[auto_1fr] lg:grid-cols-[minmax(420px,520px)_1fr] lg:grid-rows-1">
      <main className="order-2 flex flex-col bg-kumo-base px-5 pt-6 pb-5 sm:px-10 lg:order-1 lg:border-e lg:border-kumo-hairline lg:px-14 lg:pt-10">
        <header className="flex items-center justify-between gap-3">
          <Wordmark size={30} />
          <ThemeToggle />
        </header>

        <div className="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center py-10">
          <SignInForm />
        </div>

        <footer className="flex flex-col gap-4 border-t border-kumo-hairline pt-5 text-sm text-kumo-subtle sm:flex-row sm:items-center sm:justify-between">
          <p className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span>{t('lostAccess')}</span>
            <code
              dir="ltr"
              className="rounded-md bg-kumo-recessed px-1.5 py-0.5 font-mono text-tiny text-kumo-default"
            >
              karvan reset --password
            </code>
          </p>
          <LanguageSwitch className="self-start sm:self-auto" />
        </footer>
      </main>
      <div className="order-1 lg:order-2 lg:h-full">
        <LoginStage />
      </div>
    </div>
  );
}
