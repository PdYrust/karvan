import { cn } from '@cloudflare/kumo/utils';
import { locales } from '../../i18n/core';
import { useLocale, useT } from '../../i18n/locale';
import { languageNames, sharedMessages } from '../../i18n/shared';

/** Two-option switch; each name is written in its own language. */
export function LanguageSwitch({ className }: { className?: string }) {
  const { locale, setLocale } = useLocale();
  const t = useT(sharedMessages);
  return (
    <div
      role="group"
      aria-label={t('languageLabel')}
      className={cn('inline-flex rounded-lg bg-kumo-recessed p-0.5 ring ring-kumo-line', className)}
    >
      {locales.map((l) => (
        <button
          key={l}
          type="button"
          lang={l}
          aria-pressed={locale === l}
          onClick={() => setLocale(l)}
          className={cn(
            'min-h-9 rounded-md px-3 text-sm transition-colors pointer-coarse:min-h-11',
            locale === l
              ? 'bg-kumo-base font-semibold text-kumo-strong shadow-[0_1px_0_var(--color-kumo-shadow-edge)]'
              : 'text-kumo-subtle hover:text-kumo-default',
          )}
        >
          {languageNames[l]}
        </button>
      ))}
    </div>
  );
}
