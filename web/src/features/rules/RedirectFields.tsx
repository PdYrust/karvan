import { Radio } from '@cloudflare/kumo/components/radio';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { siteFormMessages } from './form-messages';
import { isPlainHttp, parseOldLink, type RuleValues } from './rule-form';
import type { FieldsProps } from './SiteFields';
import { PortField } from './PortField';
import { TextField } from './TextField';

/** Where the redirect sends the old link; a plain http address gets a warning. */
export function ToUrlField({ values, set, errorOf }: FieldsProps) {
  const t = useT(siteFormMessages);
  return (
    <>
      <TextField
        id="rule-to"
        label={t('toUrlLabel')}
        hint={<CodeText text={t('toUrlHint')} />}
        error={errorOf('toUrl')}
        value={values.toUrl}
        onChange={(toUrl) => set({ toUrl })}
        machine
        inputMode="url"
      />
      {isPlainHttp(values.toUrl) ? (
        <InlineAlert tone="warning">{t('httpTarget')}</InlineAlert>
      ) : null}
    </>
  );
}

/** A redirect's path, how it matches and its port. */
export function RedirectFields({ values, set, errorOf }: FieldsProps) {
  const t = useT(siteFormMessages);
  return (
    <fieldset className="flex flex-col gap-5">
      <div className="grid gap-5 sm:grid-cols-[1fr_10rem]">
        <TextField
          id="rule-from"
          label={t('fromPathLabel')}
          hint={<CodeText text={t('fromPathHint')} />}
          error={errorOf('fromPath')}
          value={values.fromPath}
          onChange={(fromPath) => set({ fromPath })}
          machine
        />
        <PortField id="rule-port" values={values} set={set} errorOf={errorOf} />
      </div>
      <Radio.Group
        legend={t('matchTitle')}
        value={values.match}
        onValueChange={(match: RuleValues['match']) => set({ match })}
      >
        <Radio.Item value="exact" label={t('match_exact')} />
        <Radio.Item value="prefix" label={t('match_prefix')} />
      </Radio.Group>
    </fieldset>
  );
}

/** One old link: Karvan reads its host, port and path as it is typed (D55). */
export function LinkFields({ values, set, errorOf }: FieldsProps) {
  const t = useT(siteFormMessages);
  const link = parseOldLink(values.link);
  let read = '';
  if (link) read = t('linkRead', { host: link.host, port: String(link.port), path: link.path });
  else if (values.link.trim()) read = t('linkUnreadable');
  return (
    <div className="flex flex-col gap-2">
      <TextField
        id="rule-link"
        label={t('linkLabel')}
        hint={t('linkHint')}
        // The link gives the path, the domain and the port, so their refusals show here.
        error={errorOf('fromPath') ?? errorOf('domains') ?? errorOf('port') ?? errorOf('https')}
        value={values.link}
        onChange={(text) => set({ link: text })}
        machine
        inputMode="url"
      />
      {read ? (
        <p aria-live="polite" className="text-sm text-kumo-subtle">
          <CodeText text={read} />
        </p>
      ) : null}
    </div>
  );
}
