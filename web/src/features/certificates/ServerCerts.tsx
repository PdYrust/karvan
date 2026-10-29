import { Badge } from '@cloudflare/kumo/components/badge';
import { Radio } from '@cloudflare/kumo/components/radio';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import { useFormat } from '../../lib/preferences/region';
import { domainWarningTone, sortedWarnings } from './cert-view';
import type { FoundPick, ServerCert } from './domain-form';
import { domainFormMessages } from './domain-messages';
import { certFactMessages } from './fact-messages';
import { Mono } from './parts';

// Compact radio rows in a ringed box split by hairlines, as the rule form's domain list.
const box =
  '[&>div:first-child]:sr-only [&>div:last-child]:gap-0 [&>div:last-child]:divide-y [&>div:last-child]:divide-kumo-hairline [&>div:last-child]:rounded-lg [&>div:last-child]:bg-kumo-base [&>div:last-child]:ring [&>div:last-child]:ring-kumo-hairline';
const row = 'flex w-full px-4 py-3 [&>span:last-child]:min-w-0 [&>span:last-child]:flex-1';

function CertRow({ cert, best }: { cert: ServerCert; best: boolean }) {
  const t = useT(domainFormMessages);
  const tc = useT(certFactMessages);
  const fmt = useFormat();
  const { facts } = cert;
  const stored = cert.pick.kind === 'stored';
  return (
    <span className="flex flex-col gap-0.5">
      <span className="flex items-start justify-between gap-3">
        {stored ? (
          <bdi className="min-w-0 font-medium text-kumo-strong">{cert.title}</bdi>
        ) : (
          <Mono text={cert.title} className="min-w-0 font-medium text-kumo-strong" />
        )}
        {best ? <Badge variant="outline">{t('recommended')}</Badge> : null}
      </span>
      <span className="flex flex-wrap gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
        <span>{stored ? t('storedLabel') : t('fileLabel')}</span>
        {facts.issuer ? <bdi>{t('issuedBy', { issuer: facts.issuer })}</bdi> : null}
        {facts.notAfter ? (
          <span>{t('endsOn', { date: fmt.date(new Date(facts.notAfter)) })}</span>
        ) : null}
      </span>
      {cert.covers ? null : (
        <span className="text-sm">
          <StatusLabel tone="warning">{t('notCovers')}</StatusLabel>
        </span>
      )}
      {sortedWarnings(facts.warnings).map((w) => (
        <span key={w} className="text-sm">
          <StatusLabel tone={domainWarningTone(w)}>{tc(`warn_${w}`)}</StatusLabel>
        </span>
      ))}
    </span>
  );
}

const keyOf = (p: FoundPick) =>
  p.kind === 'stored' ? `stored:${String(p.certificateId)}` : `file:${p.certFile}`;

/**
 * The certificates on this server no domain uses yet, as radio rows; one that
 * does not cover the name is listed but cannot be picked.
 */
export function ServerCerts({
  certs,
  pick,
  best,
  onPick,
}: {
  certs: readonly ServerCert[];
  pick: FoundPick | null;
  best?: string;
  onPick: (p: FoundPick) => void;
}) {
  const t = useT(domainFormMessages);
  const none = (
    <p className="rounded-lg bg-kumo-tint px-4 py-3 text-sm text-kumo-subtle">{t('serverNone')}</p>
  );
  if (certs.length === 0) return none;
  const list = (
    <Radio.Group
      legend={t('own_server')}
      value={pick ? keyOf(pick) : ''}
      onValueChange={(key: string) => {
        const c = certs.find((x) => x.key === key);
        if (c) onPick(c.pick);
      }}
      className={box}
    >
      {certs.map((c) => (
        <Radio.Item
          key={c.key}
          value={c.key}
          disabled={!c.covers}
          label={<CertRow cert={c} best={certs.length > 1 && c.key === best} />}
          className={row}
        />
      ))}
    </Radio.Group>
  );
  if (certs.some((c) => c.covers)) return list;
  return (
    <div className="flex flex-col gap-3">
      {none}
      {list}
    </div>
  );
}
