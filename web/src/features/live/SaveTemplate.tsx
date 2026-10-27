import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { Input } from '@cloudflare/kumo/components/input';
import { BookmarkSimpleIcon } from '@phosphor-icons/react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useId, useState, type FormEvent } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import { saveTemplate, templateKeys, templateNameFits } from '../../lib/api/templates';
import { errorOf, useErrorText } from '../../lib/errors/errors';
import { liveMessages } from './messages';

/** Save as template (4.6): this migration's categories, changes and fixes under a name. */
export function SaveTemplate({ jobId }: { jobId: string }) {
  const t = useT(liveMessages);
  const errorText = useErrorText();
  const qc = useQueryClient();
  const id = useId();
  const [open, setOpen] = useState(false);
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
    if (templateNameFits(name) && !save.isPending) save.mutate(name.trim());
  };
  const openDialog = () => {
    save.reset();
    setOpen(true);
  };
  // A job Karvan no longer holds (after a restart) has no plan to save.
  const failure = save.isError
    ? errorOf(save.error).code === 'not_found'
      ? t('templateGone')
      : errorText(save.error)
    : null;
  return (
    <>
      <Button
        icon={<BookmarkSimpleIcon size={16} />}
        className="pointer-coarse:h-9"
        onClick={openDialog}
      >
        {t('saveTemplate')}
      </Button>
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog className="p-6" size="base">
          <form onSubmit={submit} className="flex flex-col gap-4">
            <Dialog.Title className="text-heading font-semibold">{t('saveTemplate')}</Dialog.Title>
            <Dialog.Description className="text-kumo-subtle">
              {t('templateBody')}
            </Dialog.Description>
            <div className="flex flex-col gap-2">
              <label
                id={`${id}-label`}
                htmlFor={id}
                className="text-sm font-medium text-kumo-strong"
              >
                {t('templateName')}
              </label>
              <Input
                id={id}
                aria-labelledby={`${id}-label`}
                aria-invalid={tooLong || undefined}
                autoComplete="off"
                dir="auto"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full pointer-coarse:h-9"
              />
              <p aria-live="polite" className="text-sm text-kumo-subtle">
                {tooLong
                  ? t('templateTooLong')
                  : save.isSuccess
                    ? t('templateSaved', { name: save.data.template.name })
                    : null}
              </p>
            </div>
            {failure ? <InlineAlert>{failure}</InlineAlert> : null}
            <div className="flex flex-wrap justify-end gap-2">
              <Button className="pointer-coarse:h-9" onClick={() => setOpen(false)}>
                {t('close')}
              </Button>
              <Button
                type="submit"
                variant="primary"
                className="pointer-coarse:h-9"
                disabled={!templateNameFits(name)}
                loading={save.isPending}
              >
                {t('templateSave')}
              </Button>
            </div>
          </form>
        </Dialog>
      </Dialog.Root>
    </>
  );
}
