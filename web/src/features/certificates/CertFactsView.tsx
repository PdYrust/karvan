import { StatusLabel, type StatusTone } from '../../components/feedback/StatusBadge';
import { FactList } from '../../components/layout/FactList';
import { useT } from '../../i18n/locale';
import type { CertFacts, CertWarning } from '../../lib/api/rules';
import { useFormat } from '../../lib/preferences/region';
import { rulesMessages } from '../rules/messages';

/** A certificate that cannot serve is a danger; one that soon cannot, or misses a domain, a warning. */
const toneOf: Record<CertWarning, StatusTone> = {
  unreadable: 'danger',
  no_key: 'danger',
  key_mismatch: 'danger',
  expired: 'danger',
  not_yet_valid: 'danger',
  domain_mismatch: 'warning',
  expires_soon: 'warning',
};

/** What Karvan read from a certificate, then each problem as a status line. */
export function CertFactsView({ facts }: { facts: CertFacts }) {
  const t = useT(rulesMessages);
  const fmt = useFormat();
  const ends = facts.notAfter ? new Date(facts.notAfter) : undefined;
  const rows = facts.warnings.includes('unreadable')
    ? []
    : [
        {
          label: t('certCovers'),
          value: (
            <bdi dir="ltr" className="break-all">
              {facts.domains.join(', ')}
            </bdi>
          ),
        },
        { label: t('certIssuer'), value: <bdi>{facts.issuer}</bdi> },
        ...(ends
          ? [
              {
                label: t('certEnds'),
                value: (
                  <time dateTime={ends.toISOString()} title={fmt.dateTime(ends)}>
                    {fmt.date(ends)}
                  </time>
                ),
              },
            ]
          : []),
      ];
  return (
    <div className="flex flex-col gap-2">
      {rows.length > 0 ? <FactList facts={rows} /> : null}
      <ul className="flex flex-col gap-1.5 text-sm">
        {facts.warnings.length === 0 ? (
          <li>
            <StatusLabel tone="success">{t('certFine')}</StatusLabel>
          </li>
        ) : (
          facts.warnings.map((w) => (
            <li key={w}>
              <StatusLabel tone={toneOf[w]}>{t(`warn_${w}`)}</StatusLabel>
            </li>
          ))
        )}
      </ul>
    </div>
  );
}
