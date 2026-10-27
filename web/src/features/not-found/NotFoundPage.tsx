import { Button } from '@cloudflare/kumo/components/button';
import { CompassIcon } from '@phosphor-icons/react';
import { useNavigate } from 'react-router';
import { shellMessages } from '../../app/shell/messages';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';

/** An address inside the panel that has no page. */
export function NotFoundPage() {
  const t = useT(shellMessages);
  const navigate = useNavigate();
  usePageTitle(t('notFoundTitle'));
  return (
    <div className="flex flex-col items-center gap-3 px-6 py-20 text-center">
      <CompassIcon size={44} weight="light" aria-hidden="true" className="text-kumo-default" />
      <h1 className="text-title font-semibold text-kumo-strong">{t('notFoundTitle')}</h1>
      <p className="max-w-prose text-kumo-subtle">{t('notFoundBody')}</p>
      <Button className="mt-2 h-9 pointer-coarse:h-9" onClick={() => void navigate('/')}>
        {t('notFoundAction')}
      </Button>
    </div>
  );
}
