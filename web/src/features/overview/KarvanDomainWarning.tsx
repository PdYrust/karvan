import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import { getKarvanDomain, karvanKeys } from '../../lib/api/karvan';
import { useFormat } from '../../lib/preferences/region';
import { accessMessages } from '../certificates/access-messages';

/**
 * One line while the Karvan domain needs a look: the server's IP opened again because its
 * certificate cannot serve, or the certificate ends within 14 days and nothing renews it.
 */
export function KarvanDomainWarning() {
  const t = useT(accessMessages);
  const fmt = useFormat();
  // A server that cannot change the Karvan domain answers 404: nothing to show.
  const karvan = useQuery({ queryKey: karvanKeys.all, queryFn: getKarvanDomain });
  const k = karvan.data;
  if (!k || k.domainId < 1) return null;
  let text = '';
  if (k.access === 'fallback') text = t(`fallback_${k.reason ?? 'missing'}`);
  else if (k.endsSoon) text = t('endsSoon', { date: fmt.date(new Date(k.endsSoon)) });
  if (!text) return null;
  return (
    <InlineAlert tone="warning">
      {text}{' '}
      <Link
        to={`/domains/${String(k.domainId)}`}
        className="font-medium text-kumo-link underline-offset-4 hover:underline"
      >
        {t('openDomain')}
      </Link>
    </InlineAlert>
  );
}
