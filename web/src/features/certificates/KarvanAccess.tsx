import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import type { KarvanDomain } from '../../lib/api/karvan';
import { accessMessages } from './access-messages';

/**
 * Under the Karvan domain: that it waits for a first sign-in through it before it is the only
 * way in, or why the server's IP opens the panel again. Nothing while all is as set.
 */
export function KarvanAccess({ karvan }: { karvan: KarvanDomain }) {
  const t = useT(accessMessages);
  if (karvan.access === 'waiting') {
    return <span className="text-sm text-kumo-subtle">{t('waiting')}</span>;
  }
  if (karvan.access === 'fallback') {
    return (
      <span className="text-sm">
        <StatusLabel tone="warning">{t(`fallback_${karvan.reason ?? 'missing'}`)}</StatusLabel>
      </span>
    );
  }
  return null;
}
