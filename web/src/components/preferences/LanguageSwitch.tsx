import { cn } from '@cloudflare/kumo/utils';
import { GlobeIcon } from '@phosphor-icons/react';
import { useLocale, useT } from '../../i18n/locale';
import { languageNames, sharedMessages } from '../../i18n/shared';

/**
 * One button that switches to the other language. Its text is the other
 * language's own name, so a reader of either can find it.
 */
export function LanguageSwitch({ className }: { className?: string }) {
  const { locale, setLocale } = useLocale();
  const t = useT(sharedMessages);
  const other = locale === 'fa' ? 'en' : 'fa';
  return (
    <button
      type="button"
      lang={other}
      onClick={() => setLocale(other)}
      aria-label={`${t('languageLabel')}: ${languageNames[other]}`}
      className={cn(
        'inline-flex min-h-8 items-center gap-2 rounded-md px-2 text-sm font-medium transition-colors pointer-coarse:min-h-9',
        className ?? 'text-kumo-default hover:bg-kumo-tint',
      )}
    >
      <GlobeIcon size={16} aria-hidden="true" />
      {languageNames[other]}
    </button>
  );
}
