import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { DownloadSimpleIcon, KeyIcon } from '@phosphor-icons/react';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { reportFiles } from '../../lib/api/report';
import { useErrorText } from '../../lib/errors/errors';
import { liveMessages, type LiveKey } from './messages';

type File = 'json' | 'csv' | 'links';
const fileWords: Readonly<Record<File, LiveKey>> = {
  json: 'fileJson',
  csv: 'fileCsv',
  links: 'fileLinks',
};

/**
 * The admin password file, once (B9): a warning first, then one POST from
 * this click. Nothing is kept in the page or retried; a second try says it
 * is gone.
 */
function PasswordFile({ jobId, admin }: { jobId: string; admin: string }) {
  const t = useT(liveMessages);
  const errorText = useErrorText();
  const [open, setOpen] = useState(false);
  const file = useMutation({
    gcTime: 0,
    mutationFn: () => reportFiles.passwords(jobId),
    onSettled: () => setOpen(false),
  });
  return (
    <>
      {file.isSuccess ? (
        <p role="status" className="text-sm text-kumo-subtle">
          {t('passwordDone')}
        </p>
      ) : (
        <Button
          icon={<KeyIcon size={16} />}
          className="pointer-coarse:h-9"
          loading={file.isPending}
          onClick={() => setOpen(true)}
        >
          {t('filePassword')}
        </Button>
      )}
      {file.isError ? (
        <InlineAlert className="basis-full">{`${t('fileFailed')} ${errorText(file.error)}`}</InlineAlert>
      ) : null}
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog className="flex flex-col gap-4 p-6" size="base">
          <Dialog.Title className="text-heading font-semibold">{t('passwordTitle')}</Dialog.Title>
          <Dialog.Description className="text-kumo-subtle">
            <CodeText text={t('passwordBody', { admin })} />
          </Dialog.Description>
          <div className="flex flex-wrap justify-end gap-2">
            <Button className="pointer-coarse:h-9" onClick={() => setOpen(false)}>
              {t('cancel')}
            </Button>
            <Button
              variant="primary"
              className="pointer-coarse:h-9"
              loading={file.isPending}
              onClick={() => file.mutate()}
            >
              {t('passwordDownload')}
            </Button>
          </div>
        </Dialog>
      </Dialog.Root>
    </>
  );
}

/** A finished job's files (4.6): the report, the new links, and the admin password once. */
export function Files({
  jobId,
  links,
  password,
}: {
  jobId: string;
  /** The run moved users, so they have new links. */
  links: boolean;
  /** The admin whose generated password can be downloaded, if any. */
  password: string | null;
}) {
  const t = useT(liveMessages);
  const errorText = useErrorText();
  const file = useMutation({ mutationFn: (f: File) => reportFiles[f](jobId) });
  const files: File[] = links ? ['json', 'csv', 'links'] : ['json', 'csv'];
  return (
    <div className="flex flex-col gap-2">
      <h3 className="font-medium text-kumo-strong">{t('files')}</h3>
      <div className="flex flex-wrap items-center gap-2">
        {files.map((f) => (
          <Button
            key={f}
            icon={<DownloadSimpleIcon size={16} />}
            className="pointer-coarse:h-9"
            loading={file.isPending && file.variables === f}
            onClick={() => file.mutate(f)}
          >
            {t(fileWords[f])}
          </Button>
        ))}
        {password !== null ? <PasswordFile jobId={jobId} admin={password} /> : null}
      </div>
      {file.isError ? (
        <InlineAlert>{`${t('fileFailed')} ${errorText(file.error)}`}</InlineAlert>
      ) : null}
    </div>
  );
}
