import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { Input } from '@cloudflare/kumo/components/input';
import { Select } from '@cloudflare/kumo/components/select';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState, type FormEvent } from 'react';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Section } from '../../../components/layout/Section';
import { useT } from '../../../i18n/locale';
import {
  listTemplates,
  saveTemplate,
  templateKeys,
  templateNameFits,
} from '../../../lib/api/templates';
import { useErrorText } from '../../../lib/errors/errors';
import { applyTemplate, type StoredTemplate } from '../../../lib/plan/changes';
import { useWizard } from '../wizard-context';
import { changesMessages } from './messages';
import { useLingering } from '../../../lib/motion/use-lingering';

/** Asks before a template replaces the plan's categories, changes and fixes. */
function ApplyDialog({
  template,
  onClose,
}: {
  template: StoredTemplate | null;
  onClose: () => void;
}) {
  const t = useT(changesMessages);
  const { plan } = useWizard();
  // The dialog keeps the name while it fades out.
  const named = useLingering(template);
  return (
    <Dialog.Root open={template !== null} onOpenChange={(open) => !open && onClose()}>
      <Dialog className="flex flex-col gap-4 p-6" size="base">
        <Dialog.Title className="text-heading font-semibold">
          <bdi>{t('applyTitle', { name: named?.template.name ?? '' })}</bdi>
        </Dialog.Title>
        <Dialog.Description className="text-kumo-subtle">{t('applyBody')}</Dialog.Description>
        <div className="flex flex-wrap justify-end gap-2">
          <Button className="pointer-coarse:h-9" onClick={onClose}>
            {t('cancel')}
          </Button>
          <Button
            variant="primary"
            className="pointer-coarse:h-9"
            onClick={() => {
              if (template) plan.edit(applyTemplate(template.template));
              onClose();
            }}
          >
            {t('applyConfirm')}
          </Button>
        </div>
      </Dialog>
    </Dialog.Root>
  );
}

function SaveTemplate() {
  const t = useT(changesMessages);
  const errorText = useErrorText();
  const qc = useQueryClient();
  const { jobId, plan } = useWizard();
  const [name, setName] = useState('');
  const save = useMutation({
    mutationFn: (n: string) => saveTemplate(jobId, n),
    onSuccess: () => {
      setName('');
      void qc.invalidateQueries({ queryKey: templateKeys.all });
    },
  });
  const tooLong = name.trim() !== '' && !templateNameFits(name);
  const submit = (e: FormEvent) => {
    e.preventDefault();
    // The server saves the plan it holds, so a change still on its way would be missed.
    if (plan.saving || !templateNameFits(name)) return;
    save.mutate(name.trim());
  };
  return (
    <form onSubmit={submit} className="flex flex-col gap-2 border-t border-kumo-hairline pt-4">
      <label
        id="template-name-label"
        htmlFor="template-name"
        className="text-sm font-medium text-kumo-strong"
      >
        {t('saveLabel')}
      </label>
      <div className="flex flex-wrap gap-2">
        <Input
          id="template-name"
          aria-labelledby="template-name-label"
          aria-invalid={tooLong || undefined}
          autoComplete="off"
          dir="auto"
          placeholder={t('saveName')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full sm:w-72 pointer-coarse:h-9"
        />
        <Button
          type="submit"
          disabled={!templateNameFits(name) || plan.saving || save.isPending}
          className="pointer-coarse:h-9"
        >
          {t('save')}
        </Button>
      </div>
      <p aria-live="polite" className="text-sm text-kumo-subtle">
        {tooLong
          ? t('nameTooLong')
          : plan.saving && name.trim()
            ? t('saveWait')
            : save.isSuccess
              ? t('saved', { name: save.data.template.name })
              : null}
      </p>
      {save.isError ? <InlineAlert>{errorText(save.error)}</InlineAlert> : null}
    </form>
  );
}

/** Apply a saved template to this plan, or save this plan's choices as one. */
export function Templates() {
  const t = useT(changesMessages);
  const list = useQuery({ queryKey: templateKeys.all, queryFn: listTemplates });
  const [picked, setPicked] = useState('');
  const [asking, setAsking] = useState<StoredTemplate | null>(null);
  const templates = list.data ?? [];
  const chosen = templates.find((x) => String(x.id) === picked);
  return (
    <Section title={t('templatesTitle')}>
      <div className="flex flex-col gap-4 px-4 pb-4">
        <p className="text-kumo-subtle">{t('templatesIntro')}</p>
        {list.isError ? (
          <ErrorState
            message={t('templatesFailed')}
            error={list.error}
            onRetry={() => void list.refetch()}
          />
        ) : list.isPending ? null : templates.length === 0 ? (
          <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('noTemplates')}</p>
        ) : (
          <div className="flex flex-col gap-2">
            <span id="template-pick" className="text-sm font-medium text-kumo-strong">
              {t('templateLabel')}
            </span>
            <div className="flex gap-2">
              <Select
                aria-labelledby="template-pick"
                placeholder={t('choose')}
                className="min-w-0 flex-1 sm:w-72 sm:flex-none pointer-coarse:h-9"
                value={picked || null}
                items={Object.fromEntries(templates.map((x) => [String(x.id), x.template.name]))}
                onValueChange={(id) => setPicked(id ?? '')}
              />
              <Button
                disabled={!chosen}
                className="pointer-coarse:h-9"
                onClick={() => setAsking(chosen ?? null)}
              >
                {t('applyTemplate')}
              </Button>
            </div>
          </div>
        )}
        <SaveTemplate />
      </div>
      <ApplyDialog template={asking} onClose={() => setAsking(null)} />
    </Section>
  );
}
