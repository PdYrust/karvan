import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import { overviewMessages } from './messages';
import { useOverview } from './use-overview';

/** The same address over HTTPS: every panel port answers both. */
function httpsHere(): string {
  const url = new URL(window.location.href);
  url.protocol = 'https:';
  return url.toString();
}

/** One line while this page came over plain HTTP, with a link to it over HTTPS. */
export function PlainHttpWarning() {
  const t = useT(overviewMessages);
  const overview = useOverview();
  if (overview.data?.security.tls !== 'off') return null;
  return (
    <InlineAlert>
      {t('plainHttp')}{' '}
      <a
        href={httpsHere()}
        className="font-medium text-kumo-link underline-offset-4 hover:underline"
      >
        {t('openHttps')}
      </a>
    </InlineAlert>
  );
}
