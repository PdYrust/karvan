import { Link } from 'react-router';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import { overviewMessages } from './messages';
import { useOverview } from './use-overview';

/** One line while the panel is served over plain HTTP, with the way to the HTTPS tab. */
export function PlainHttpWarning() {
  const t = useT(overviewMessages);
  const overview = useOverview();
  if (overview.data?.security.tls !== 'off') return null;
  return (
    <InlineAlert>
      {t('plainHttp')}{' '}
      <Link
        to="/settings/https"
        className="font-medium text-kumo-link underline-offset-4 hover:underline"
      >
        {t('turnOnHttps')}
      </Link>
    </InlineAlert>
  );
}
