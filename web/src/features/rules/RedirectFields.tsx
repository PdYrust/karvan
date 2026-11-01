import { Button } from '@cloudflare/kumo/components/button';
import { Input } from '@cloudflare/kumo/components/input';
import { cn } from '@cloudflare/kumo/utils';
import { PlusIcon } from '@phosphor-icons/react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { TrashIcon } from '../../components/icons/regular';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { siteFormMessages } from './form-messages';
import { PortField } from './PortField';
import { isPlainHttp, normalizeTargetUrl } from './rule-form';
import type { FieldsProps } from './SiteFields';
import { TextField } from './TextField';

/** Where the custom redirect sends matching requests; a plain http address gets a warning. */
export function ToUrlField({ values, set, errorOf }: FieldsProps) {
  const t = useT(siteFormMessages);
  const normalized = normalizeTargetUrl(values.toUrl);
  return (
    <>
      <TextField
        id="rule-to"
        label={t('toUrlLabel')}
        hint={<CodeText text={t('toUrlHint')} />}
        error={errorOf('toUrl')}
        value={values.toUrl}
        onChange={(toUrl) => set({ toUrl })}
        placeholder="https://new.example.com:2096"
        machine
        inputMode="url"
      />
      {isPlainHttp(normalized) ? <InlineAlert tone="warning">{t('httpTarget')}</InlineAlert> : null}
    </>
  );
}

/** A custom rule's port, allowed source paths (multiple) and destination target. */
export function RedirectFields({ values, set, errorOf }: FieldsProps) {
  const t = useT(siteFormMessages);
  const paths = values.fromPaths.length > 0 ? values.fromPaths : [''];
  const pathError = errorOf('fromPath');
  const setPath = (index: number, val: string) => {
    const next = paths.map((p, i) => (i === index ? val : p));
    set({ fromPaths: next, fromPath: next[0] ?? '' });
  };
  const removePath = (index: number) => {
    const next = paths.filter((_, i) => i !== index);
    set({ fromPaths: next.length > 0 ? next : [''], fromPath: next[0] ?? '' });
  };
  const addPath = () => set({ fromPaths: [...paths, ''] });
  return (
    <div className="flex flex-col gap-6">
      <div className="max-w-[12rem]">
        <PortField id="rule-port" values={values} set={set} errorOf={errorOf} />
      </div>
      <fieldset className="flex flex-col gap-3">
        <legend id="rule-paths-label" className="mb-0.5 flex flex-col gap-1">
          <span className="font-medium text-kumo-strong">{t('fromPathLabel')}</span>
          <span id="rule-paths-hint" className="text-sm text-kumo-subtle">
            <CodeText text={t('fromPathHint')} />
          </span>
        </legend>
        <div className="flex flex-col gap-2">
          {paths.map((p, i) => (
            <div key={i} className="flex items-center gap-2">
              <div className="flex-1">
                <Input
                  id={`rule-path-${String(i)}`}
                  name={`rule-path-${String(i)}`}
                  value={p}
                  placeholder={i === 0 ? 'sub' : 'dashboard'}
                  dir="ltr"
                  autoComplete="off"
                  autoCapitalize="off"
                  spellCheck={false}
                  aria-labelledby="rule-paths-label"
                  aria-describedby={
                    pathError ? 'rule-paths-hint rule-paths-error' : 'rule-paths-hint'
                  }
                  aria-invalid={pathError ? true : undefined}
                  className={cn(
                    'h-10 w-full font-mono pointer-coarse:h-9',
                    pathError && 'ring-kumo-danger',
                  )}
                  onChange={(e) => setPath(i, e.target.value)}
                />
              </div>
              {paths.length > 1 ? (
                <Button
                  type="button"
                  variant="ghost"
                  shape="square"
                  aria-label={t('removePath', { n: i + 1 })}
                  title={t('removePath', { n: i + 1 })}
                  icon={<TrashIcon size={16} />}
                  className="shrink-0 pointer-coarse:size-9"
                  onClick={() => removePath(i)}
                />
              ) : null}
            </div>
          ))}
        </div>
        {pathError ? (
          <p id="rule-paths-error" className="text-sm text-kumo-danger">
            <CodeText text={pathError} />
          </p>
        ) : null}
        <div>
          <Button
            type="button"
            icon={<PlusIcon size={16} />}
            className="pointer-coarse:h-9"
            onClick={addPath}
          >
            {t('addPath')}
          </Button>
        </div>
      </fieldset>
      <ToUrlField values={values} set={set} errorOf={errorOf} />
    </div>
  );
}
