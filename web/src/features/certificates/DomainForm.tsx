import { Button } from '@cloudflare/kumo/components/button';
import { Radio } from '@cloudflare/kumo/components/radio';
import { CaretDownIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { Certificate } from '../../lib/api/certificates';
import type { CertChoice } from '../../lib/api/domains';
import { useAcmeErrorText } from '../../lib/errors/acme';
import type { CertMode } from '../rules/site-form';
import { TextField } from '../rules/TextField';
import { AcmeFields } from './AcmeFields';
import { ModeFields } from './CertFields';
import {
  acmeMethodsFor,
  canSubmit,
  choiceOf,
  isPortError,
  mainWays,
  refusedDomainField,
  type DomainFormValues,
  type Way,
} from './domain-form';
import { domainFormMessages } from './domain-messages';
import { FoundChoices } from './FoundChoices';

const modeOf: Partial<Record<Way, CertMode>> = {
  upload: 'upload',
  paste: 'paste',
  none: 'none',
  file: 'files',
};

/** The ways as radio cards; the file path waits under More, for experts. */
function WayChooser({ way, onWay }: { way: Way; onWay: (w: Way) => void }) {
  const t = useT(domainFormMessages);
  const [more, setMore] = useState(way === 'file');
  const ways = more ? [...mainWays, 'file' as const] : mainWays;
  return (
    <div className="flex flex-col gap-2">
      <Radio.Group
        legend={t('certTitle')}
        appearance="card"
        value={way}
        onValueChange={(w: Way) => onWay(w)}
        className="w-full"
      >
        {ways.map((w) => (
          <Radio.Item
            key={w}
            value={w}
            label={
              <span className="flex flex-col gap-0.5">
                <span className="font-medium text-kumo-strong">{t(`way_${w}`)}</span>
                <span className="text-sm text-kumo-subtle">{t(`wayHint_${w}`)}</span>
              </span>
            }
          />
        ))}
      </Radio.Group>
      {more ? null : (
        <Button
          variant="ghost"
          icon={<CaretDownIcon size={16} />}
          className="self-start pointer-coarse:h-9"
          onClick={() => setMore(true)}
        >
          {t('moreWays')}
        </Button>
      )}
    </div>
  );
}

/** Add domain, or Change certificate when `isNew` is false: the name, one way, its fields. */
export function DomainForm({
  initial,
  isNew,
  certificates,
  needsAgreement,
  saving,
  error,
  onSave,
  onReset,
  onCancel,
}: {
  initial: DomainFormValues;
  isNew: boolean;
  certificates: readonly Certificate[];
  needsAgreement: boolean;
  saving: boolean;
  error: unknown;
  onSave: (name: string, choice: CertChoice) => void;
  onReset: () => void;
  onCancel: () => void;
}) {
  const t = useT(domainFormMessages);
  const errorText = useAcmeErrorText();
  const [v, setV] = useState(initial);
  const set = (patch: Partial<DomainFormValues>) => setV((old) => ({ ...old, ...patch }));
  const refused = refusedDomainField(error);
  const reason = error ? errorText(error) : '';
  const errorOf = (f: string) => (refused === f ? reason : undefined);
  const mode = modeOf[v.way];
  const setName = (name: string) => {
    // A wildcard gets its certificate only by DNS, so the method follows the name.
    const methods = acmeMethodsFor(name);
    set({ name, method: methods.includes(v.method) ? v.method : 'dns-01' });
  };
  return (
    <form
      noValidate
      className="flex max-w-[720px] flex-col gap-8"
      onSubmit={(e) => {
        e.preventDefault();
        const choice = choiceOf(v, needsAgreement);
        if (choice) onSave(v.name, choice);
      }}
    >
      {isNew ? (
        <TextField
          id="domain-name"
          label={t('nameLabel')}
          hint={<CodeText text={t('nameHint')} />}
          error={errorOf('name')}
          value={v.name}
          onChange={setName}
          placeholder="sub.example.com"
          machine
        />
      ) : null}
      <fieldset className="flex flex-col gap-5">
        <WayChooser way={v.way} onWay={(way) => set({ way })} />
        {v.way === 'found' ? (
          <FoundChoices
            name={v.name}
            certificates={certificates}
            pick={v.pick}
            onPick={(pick) => set({ pick })}
          />
        ) : null}
        {v.way === 'acme' ? (
          <AcmeFields
            name={v.name}
            method={v.method}
            agreeTos={v.agreeTos}
            needsAgreement={needsAgreement}
            set={set}
          />
        ) : null}
        {mode ? (
          <ModeFields
            values={{ ...v, certMode: mode }}
            set={set}
            errorOf={errorOf}
            hasKey={false}
          />
        ) : null}
      </fieldset>
      {error && !refused ? (
        <InlineAlert>
          <span className="flex flex-col items-start gap-3">
            <CodeText text={`${t('failed')} ${reason}`} />
            {isPortError(error) ? (
              <Button
                className="pointer-coarse:h-9"
                onClick={() => {
                  set({ way: 'acme', method: 'dns-01' });
                  onReset();
                }}
              >
                {t('useDns')}
              </Button>
            ) : null}
          </span>
        </InlineAlert>
      ) : null}
      <div className="flex flex-wrap justify-end gap-2">
        <Button type="button" className="pointer-coarse:h-9" onClick={onCancel}>
          {t('cancel')}
        </Button>
        <Button
          type="submit"
          variant="primary"
          loading={saving}
          disabled={!canSubmit(v, isNew, needsAgreement)}
          className="pointer-coarse:h-9"
        >
          {isNew ? t('addSubmit') : t('save')}
        </Button>
      </div>
    </form>
  );
}
