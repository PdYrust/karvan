import { Button } from '@cloudflare/kumo/components/button';
import type { ReactNode } from 'react';
import { Bone } from '../../components/feedback/PageSkeleton';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { TextField } from '../rules/TextField';
import { ChoiceList } from './ChoiceList';
import { isWildcard, stepAfter, type Https, type Source, type Step, type Way } from './domain-form';
import { domainFormMessages } from './domain-messages';

export const action = 'flex-1 justify-center sm:flex-none pointer-coarse:h-9';

/** The bar at the bottom of a step: a note at the start, the buttons at the end. */
export function Footer({ note, children }: { note?: ReactNode; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-3 border-t border-kumo-hairline pt-5 sm:flex-row sm:items-center sm:justify-end">
      {note}
      <div className="flex gap-2 sm:justify-end">{children}</div>
    </div>
  );
}

const mark = '\u0000';

/** The HTTPS question's heading, the domain kept left to right inside it. */
function HttpsTitle({ name }: { name: string }) {
  const t = useT(domainFormMessages);
  const [before, after] = t('httpsTitle', { name: mark }).split(mark);
  return (
    <>
      {before}
      <bdi dir="ltr">{name.trim()}</bdi>
      {after}
    </>
  );
}

/** The heading of a step after the first: the question it asks, or the way chosen. */
export function StepTitle({ step, way, name }: { step: Step; way: Way; name: string }) {
  const t = useT(domainFormMessages);
  if (step === 'https') return <HttpsTitle name={name} />;
  if (step === 'source') return t('sourceTitle');
  if (way === 'none') return t('https_without');
  if (way === 'acme-http' || way === 'acme-dns') return t(`source_${way}`);
  return t('ownTitle');
}

/** With a certificate or without; a press goes on. */
export function HttpsQuestion({
  chosen,
  onChoose,
}: {
  chosen?: Https;
  onChoose: (h: Https) => void;
}) {
  const t = useT(domainFormMessages);
  return (
    <ChoiceList<Https>
      label={t('step_https')}
      chosen={chosen}
      rows={[
        {
          value: 'with',
          title: t('https_with'),
          hint: t('httpsHint_with'),
          badge: t('recommended'),
        },
        { value: 'without', title: t('https_without'), hint: t('httpsHint_without') },
      ]}
      onChoose={onChoose}
    />
  );
}

/**
 * A new certificate from Let's Encrypt by either check, or one the operator
 * has, the recommended one marked. Skeleton rows while Karvan looks on this server.
 */
export function SourceQuestion({
  name,
  ready,
  found,
  recommended,
  chosen,
  onChoose,
}: {
  name: string;
  ready: boolean;
  /** How many certificates on this server cover the name. */
  found: number;
  recommended: Source;
  chosen?: Source;
  onChoose: (s: Source) => void;
}) {
  const t = useT(domainFormMessages);
  if (!ready) {
    return (
      <div role="status" aria-label={t('scanning')} className="flex flex-col gap-2">
        <Bone className="h-16 w-full" />
        <Bone className="h-16 w-full" />
        <Bone className="h-16 w-full" />
      </div>
    );
  }
  const wildcard = isWildcard(name);
  const hint = (s: Source) => {
    if (s === 'acme-http' && wildcard) return t('wildcardDns');
    if (s === 'own' && found > 0) return t('ownFound', { count: found });
    return t(`sourceHint_${s}`);
  };
  return (
    <ChoiceList<Source>
      label={t('sourceTitle')}
      chosen={chosen}
      rows={(['acme-http', 'acme-dns', 'own'] as const).map((s) => ({
        value: s,
        title: t(`source_${s}`),
        hint: hint(s),
        badge: s === recommended ? t('recommended') : undefined,
        disabled: s === 'acme-http' && wildcard,
      }))}
      onChoose={onChoose}
    />
  );
}

/** The first step: the domain, with its helper line, and Continue (Enter too). */
export function NameStep({
  name,
  error,
  onName,
  onContinue,
}: {
  name: string;
  error?: string;
  onName: (name: string) => void;
  onContinue: () => void;
}) {
  const t = useT(domainFormMessages);
  const ready = stepAfter(name) === 'https';
  return (
    <form
      noValidate
      className="flex flex-col gap-8"
      onSubmit={(e) => {
        e.preventDefault();
        if (ready) onContinue();
      }}
    >
      <TextField
        id="domain-name"
        label={t('nameLabel')}
        hint={<CodeText text={t('nameHint')} />}
        error={error}
        value={name}
        onChange={onName}
        placeholder="sub.example.com"
        machine
      />
      <Footer>
        <Button
          type="submit"
          variant="primary"
          disabled={!ready}
          className="w-full justify-center sm:w-auto pointer-coarse:h-9"
        >
          {t('continue')}
        </Button>
      </Footer>
    </form>
  );
}
