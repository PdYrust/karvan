import { Button } from '@cloudflare/kumo/components/button';
import { Input } from '@cloudflare/kumo/components/input';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { ArrowCounterClockwiseIcon, PencilSimpleIcon, TrashIcon } from '@phosphor-icons/react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { useNavigate } from 'react-router';
import { dialogMessages } from '../../components/dialogs/messages';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { FormField } from '../../components/form/FormField';
import { useT } from '../../i18n/locale';
import {
  batchKeys,
  deleteBatch,
  reactivateBatch,
  renameBatch,
  type Batch,
} from '../../lib/api/batches';
import { migrationKeys } from '../../lib/api/migrations';
import { batchErrorMessages, batchErrorText } from '../../lib/errors/batches';
import { useErrorText } from '../../lib/errors/errors';
import { BatchDialog } from './BatchDialog';
import { batchActions } from './batch-view';
import { batchesMessages } from './messages';

const button = 'pointer-coarse:h-9';

/** The failed request's sentence: the batch codes, else the shared words. */
function useFailure() {
  const tb = useT(batchErrorMessages);
  const shared = useErrorText();
  return (lead: string, err: unknown) => `${lead} ${batchErrorText(tb, shared, err)}`;
}

/** Rename, delete and make active again: the header buttons and their dialogs (D8). */
export function useBatchActions(batch: Batch) {
  const t = useT(batchesMessages);
  const td = useT(dialogMessages);
  const failure = useFailure();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const id = useId();
  const [renaming, setRenaming] = useState(false);
  const [name, setName] = useState('');
  const [deleting, setDeleting] = useState(false);
  // Every list that shows the batch reads it again, the Migrations list too.
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: batchKeys.all });
    void qc.invalidateQueries({ queryKey: migrationKeys.all });
  };
  const rename = useMutation({
    mutationFn: (n: string) => renameBatch(batch.id, n),
    onSuccess: (b) => {
      setRenaming(false);
      qc.setQueryData(batchKeys.batch(b.id), b);
      toasts.add({ title: t('renamed') });
      refresh();
    },
  });
  const remove = useMutation({
    mutationFn: (confirm: string) => deleteBatch(batch.id, confirm),
    onSuccess: () => {
      setDeleting(false);
      toasts.add({ title: t('deleted') });
      // The page still shows the deleted batch until it leaves, so its own
      // queries are left alone: read again, they would only answer 404.
      const own = batchKeys.batch(batch.id);
      void qc.invalidateQueries({
        queryKey: batchKeys.all,
        predicate: (q) => !own.every((part, i) => q.queryKey[i] === part),
      });
      void qc.invalidateQueries({ queryKey: migrationKeys.all });
      void navigate('/batches');
    },
  });
  const reactivate = useMutation({
    mutationFn: () => reactivateBatch(batch.id),
    onSuccess: (b) => {
      qc.setQueryData(batchKeys.batch(b.id), b);
      toasts.add({ title: t('reactivated') });
      refresh();
    },
  });
  const can = batchActions(batch);
  const actions = (
    <>
      <Button
        icon={<PencilSimpleIcon size={16} />}
        className={button}
        onClick={() => {
          rename.reset();
          setName(batch.name);
          setRenaming(true);
        }}
      >
        {t('rename')}
      </Button>
      {can.remove ? (
        <Button
          icon={<TrashIcon size={16} />}
          className={button}
          onClick={() => {
            remove.reset();
            setDeleting(true);
          }}
        >
          {t('delete')}
        </Button>
      ) : null}
      {can.reactivate ? (
        <Button
          variant="primary"
          icon={<ArrowCounterClockwiseIcon size={16} />}
          className={button}
          loading={reactivate.isPending}
          onClick={() => reactivate.mutate()}
        >
          {t('reactivate')}
        </Button>
      ) : null}
    </>
  );
  const trimmed = name.trim();
  const dialogs = (
    <>
      <BatchDialog
        open={renaming}
        onOpenChange={setRenaming}
        title={t('renameTitle')}
        busy={rename.isPending}
        onSubmit={() => trimmed && rename.mutate(trimmed)}
        footer={
          <>
            <Button
              className={button}
              disabled={rename.isPending}
              onClick={() => setRenaming(false)}
            >
              {td('cancel')}
            </Button>
            <Button
              type="submit"
              variant="primary"
              className={button}
              disabled={!trimmed}
              loading={rename.isPending}
            >
              {t('save')}
            </Button>
          </>
        }
      >
        {rename.isError ? (
          <InlineAlert>{failure(t('renameFailed'), rename.error)}</InlineAlert>
        ) : null}
        <FormField id={id} label={t('nameLabel')}>
          <Input
            id={id}
            aria-labelledby={`${id}-label`}
            value={name}
            maxLength={64}
            autoComplete="off"
            disabled={rename.isPending}
            onChange={(e) => setName(e.target.value)}
            className="h-10 pointer-coarse:h-9"
          />
        </FormField>
      </BatchDialog>
      <TypeToConfirm
        open={deleting}
        onOpenChange={setDeleting}
        title={t('deleteTitle')}
        word={batch.name}
        action={t('delete')}
        pending={remove.isPending}
        error={remove.isError ? failure(t('deleteFailed'), remove.error) : undefined}
        onConfirm={() => remove.mutate(batch.name)}
      >
        {t('deleteBody')}
      </TypeToConfirm>
    </>
  );
  const failed = reactivate.isError ? failure(t('reactivateFailed'), reactivate.error) : undefined;
  return { actions, dialogs, failed };
}
