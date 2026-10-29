import { Badge } from '@cloudflare/kumo/components/badge';
import { Radio } from '@cloudflare/kumo/components/radio';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import { useFormat } from '../../lib/preferences/region';
import { domainWarningTone, sortedWarnings } from './cert-view';
import type { FoundChoice, FoundPick, Way } from './domain-form';
import { domainFormMessages, type DomainFormKey } from './domain-messages';
import { certFactMessages } from './fact-messages';
import { Mono } from './parts';

// Compact rows in a ringed box split by hairlines, as the rule form's domain list.
const box =
  '[&>div:last-child]:gap-0 [&>div:last-child]:divide-y [&>div:last-child]:divide-kumo-hairline [&>div:last-child]:rounded-lg [&>div:last-child]:bg-kumo-base [&>div:last-child]:ring [&>div:last-child]:ring-kumo-hairline [&>div:first-child]:sr-only';
const row = 'flex w-full px-4 py-3 [&>span:last-child]:min-w-0 [&>span:last-child]:flex-1';

function FoundRow({ choice, best }: { choice: FoundChoice; best: boolean }) {
  const t = useT(domainFormMessages);
  const tc = useT(certFactMessages);
  const fmt = useFormat();
  const { facts } = choice;
  const stored = choice.pick.kind === 'stored';
  return (
    <span className="flex flex-col gap-0.5">
      <span className="flex items-start justify-between gap-3">
        {stored ? (
          <bdi className="min-w-0 font-medium text-kumo-strong">{choice.title}</bdi>
        ) : (
          <Mono text={choice.title} className="min-w-0 font-medium text-kumo-strong" />
        )}
        {best ? <Badge variant="outline">{t('recommended')}</Badge> : null}
      </span>
      <span className="flex flex-wrap gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
        <span>{stored ? t('storedLabel') : t('way_found')}</span>
        {facts.issuer ? <bdi>{t('issuedBy', { issuer: facts.issuer })}</bdi> : null}
        {facts.notAfter ? (
          <span>{t('endsOn', { date: fmt.date(new Date(facts.notAfter)) })}</span>
        ) : null}
      </span>
      {sortedWarnings(facts.warnings).map((w) => (
        <span key={w} className="text-sm">
          <StatusLabel tone={domainWarningTone(w)}>{tc(`warn_${w}`)}</StatusLabel>
        </span>
      ))}
    </span>
  );
}

type OneWay = Exclude<Way, 'found'>;
const hints: Partial<Record<OneWay, DomainFormKey>> = {
  'acme-http': 'wayHint_acme-http',
  'acme-dns': 'wayHint_acme-dns',
  upload: 'wayHint_upload',
  paste: 'wayHint_paste',
  file: 'wayHint_file',
};

function WayRow({ way }: { way: OneWay }) {
  const t = useT(domainFormMessages);
  const hint = hints[way];
  return (
    <span className="flex flex-col gap-0.5">
      <span className="font-medium text-kumo-strong">{t(`way_${way}`)}</span>
      {hint ? <span className="text-sm text-kumo-subtle">{t(hint)}</span> : null}
    </span>
  );
}

const keyOf = (p: FoundPick) =>
  p.kind === 'stored' ? `stored:${String(p.certificateId)}` : `file:${p.certFile}`;

/**
 * The chosen answer: the certificates that cover the name as radio rows, the
 * best one marked when there are several, or the one way picked instead.
 */
export function CertChoices({
  way,
  choices,
  pick,
  best,
  onPick,
}: {
  way: Way;
  choices: readonly FoundChoice[];
  pick: FoundPick | null;
  /** The key of the recommended certificate. */
  best?: string;
  onPick: (p: FoundPick) => void;
}) {
  const t = useT(domainFormMessages);
  if (way !== 'found') {
    return (
      <Radio.Group legend={t('certTitle')} value={way} className={box}>
        <Radio.Item value={way} label={<WayRow way={way} />} className={row} />
      </Radio.Group>
    );
  }
  return (
    <Radio.Group
      legend={t('certTitle')}
      value={pick ? keyOf(pick) : ''}
      onValueChange={(key: string) => {
        const c = choices.find((x) => x.key === key);
        if (c) onPick(c.pick);
      }}
      className={box}
    >
      {choices.map((c) => (
        <Radio.Item
          key={c.key}
          value={c.key}
          label={<FoundRow choice={c} best={choices.length > 1 && c.key === best} />}
          className={row}
        />
      ))}
    </Radio.Group>
  );
}
