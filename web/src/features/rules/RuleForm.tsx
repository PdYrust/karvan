import { Button } from '@cloudflare/kumo/components/button';
import { Radio } from '@cloudflare/kumo/components/radio';
import { useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { SwitchRow } from '../../components/form/SwitchRow';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { Batch } from '../../lib/api/batches';
import { ApiError } from '../../lib/api/client';
import type { Domain } from '../../lib/api/domains';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { DestinationField } from '../wizard/connect/DestinationField';
import { BatchPicker } from './BatchPicker';
import { AddFirst, DomainPicker, HttpsField } from './DomainPicker';
import { siteFormMessages } from './form-messages';
import { LinkFields, RedirectFields, ToUrlField } from './RedirectFields';
import {
  batchDefaults,
  canSubmit,
  isPlainHttp,
  parseOldLink,
  refusedField,
  ruleKinds,
  unregistered,
  withBatches,
  withDomains,
  withLink,
  type RuleField,
  type RuleKind,
  type RuleValues,
} from './rule-form';
import { AnswerFields, OldPanelFields } from './SiteFields';
import { TextField } from './TextField';

/** The four kinds as radio cards; a saved rule keeps its kind, so only Add a rule has them. */
function KindChooser({ kind, onKind }: { kind: RuleKind; onKind: (k: RuleKind) => void }) {
  const t = useT(siteFormMessages);
  return (
    <Radio.Group
      legend={t('kindTitle')}
      appearance="card"
      value={kind}
      onValueChange={(k: RuleKind) => onKind(k)}
      className="w-full"
    >
      {ruleKinds.map((k) => (
        <Radio.Item
          key={k}
          value={k}
          label={
            <span className="flex flex-col gap-0.5">
              <span className="font-medium text-kumo-strong">{t(`kind_${k}`)}</span>
              <span className="text-sm text-kumo-subtle">{t(`kind_${k}_hint`)}</span>
            </span>
          }
        />
      ))}
    </Radio.Group>
  );
}

/**
 * Add a rule or change one: its kind, its name and the kind's fields, the
 * registered domains and HTTPS. A refused field gets its sentence under it.
 */
export function RuleForm({
  initial,
  isNew,
  domains,
  batches,
  saving,
  error,
  onSave,
  onCancel,
}: {
  initial: RuleValues;
  isNew: boolean;
  domains: readonly Domain[];
  batches: readonly Batch[];
  saving: boolean;
  error: unknown;
  onSave: (v: RuleValues, picked: Batch[]) => void;
  onCancel: () => void;
}) {
  const t = useT(siteFormMessages);
  const errorText = useSubscriptionErrorText();
  // A new rule from a batch starts with what that batch knows.
  const [v, setV] = useState(() =>
    isNew && initial.batchIds.length > 0
      ? withBatches(initial, initial.batchIds, batches, domains)
      : initial,
  );
  const set = (patch: Partial<RuleValues>) => setV((old) => ({ ...old, ...patch }));
  const picked = batches.filter((b) => v.batchIds.includes(b.id));
  const facts = batchDefaults(picked, domains);
  const refused = refusedField(error);
  const reasoned = error instanceof ApiError && typeof error.detail.reason === 'string';
  const errorOf = (f: RuleField) =>
    refused === f ? (reasoned ? errorText(error) : t(`field_${f}`)) : undefined;
  const fields = { values: v, set, errorOf };
  const panelRule = v.kind === 'batches' || v.kind === 'manual';
  const link = v.kind === 'link' ? parseOldLink(v.link) : undefined;
  const plainSource =
    v.kind === 'manual'
      ? isPlainHttp(v.sourceUrl)
      : picked.some((b) => isPlainHttp(b.source.origin));
  const domainField = (
    <DomainPicker
      domains={domains}
      picked={v.domains}
      missing={v.kind === 'batches' ? unregistered(facts.hosts, domains) : []}
      https={v.https}
      error={errorOf('domains') ?? errorOf('https')}
      onPick={(names) => setV((old) => withDomains(old, names, domains))}
      onHttps={(https) => set({ https })}
    />
  );
  return (
    <form
      noValidate
      className="flex max-w-[720px] flex-col gap-8"
      onSubmit={(e) => {
        e.preventDefault();
        onSave(v, picked);
      }}
    >
      {isNew ? <KindChooser kind={v.kind} onKind={(kind) => set({ kind })} /> : null}
      <TextField
        id="site-name"
        label={t('nameLabel')}
        hint={t('nameHint')}
        error={errorOf('name')}
        value={v.name}
        onChange={(name) => set({ name })}
      />
      {v.kind === 'batches' ? (
        <>
          <BatchPicker
            batches={batches}
            picked={v.batchIds}
            error={errorOf('batchIds')}
            onPick={(ids) =>
              isNew
                ? setV((old) => withBatches(old, ids, batches, domains))
                : set({ batchIds: ids })
            }
          />
          <AnswerFields {...fields} path={!facts.oldPath} />
          {v.sourceIp || refused === 'sourceIp' ? (
            <TextField
              id="site-ip"
              label={t('ipLabel')}
              hint={t('ipHint')}
              error={errorOf('sourceIp')}
              value={v.sourceIp}
              onChange={(sourceIp) => set({ sourceIp })}
              machine
            />
          ) : null}
        </>
      ) : null}
      {v.kind === 'manual' ? (
        <>
          <div className="flex flex-col gap-2">
            <DestinationField value={v.panelId} onChange={(panelId) => set({ panelId })} />
            {errorOf('panelId') ? (
              <p className="text-sm text-kumo-danger">{errorOf('panelId')}</p>
            ) : null}
          </div>
          <OldPanelFields {...fields} />
          <AnswerFields {...fields} />
        </>
      ) : null}
      {v.kind === 'redirect' ? <RedirectFields {...fields} /> : null}
      {v.kind === 'link' ? (
        <div className="flex flex-col gap-5">
          <LinkFields
            {...fields}
            set={(p) => setV((old) => withLink(old, p.link ?? '', domains))}
          />
          {link && v.domains.length === 0 ? (
            <ul className="rounded-lg bg-kumo-base ring ring-kumo-hairline">
              <AddFirst host={link.host} />
            </ul>
          ) : null}
          {link && v.domains.length > 0 ? (
            <HttpsField
              domains={domains}
              picked={v.domains}
              https={v.https}
              onHttps={(https) => set({ https })}
            />
          ) : null}
        </div>
      ) : (
        domainField
      )}
      {v.kind === 'redirect' || v.kind === 'link' ? <ToUrlField {...fields} /> : null}
      {panelRule ? (
        <div className="flex flex-col gap-5">
          <SwitchRow
            id="site-lazy"
            label={t('lazyLabel')}
            hint={t('lazyHint')}
            checked={v.lazy}
            onChange={(lazy) => set({ lazy })}
          />
          {plainSource ? (
            <SwitchRow
              id="site-plain"
              label={t('plainLabel')}
              hint={t('plainHint')}
              checked={v.plainHttpOk}
              onChange={(plainHttpOk) => set({ plainHttpOk })}
            />
          ) : null}
        </div>
      ) : null}
      {error && !refused ? (
        <InlineAlert>
          <CodeText text={`${t('saveFailed')} ${errorText(error)}`} />
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
          disabled={!canSubmit(v, domains)}
          className="pointer-coarse:h-9"
        >
          {isNew ? t('addSite') : t('save')}
        </Button>
      </div>
    </form>
  );
}
