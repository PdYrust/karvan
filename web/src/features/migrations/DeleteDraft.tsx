import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { DotsThreeIcon, TrashIcon } from '@phosphor-icons/react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { dialogMessages } from '../../components/dialogs/messages';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { PanelLabel } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import { deleteJob, migrationKeys, originHost, type JobRow } from '../../lib/api/migrations';
import { useErrorText } from '../../lib/errors/errors';
import { isDraft } from './job-place';
import { JobMove } from './JobLabels';
import { listMessages } from './list-messages';

export type DraftJob = Pick<JobRow, 'id' | 'state' | 'source' | 'panel'>;

const jobName = (job: DraftJob) => job.source.admin || originHost(job.source.origin);

/** A job's "..." menu. Only a draft has something to offer (Delete draft), so other jobs get none. */
export function JobMenu({ job, onDelete }: { job: DraftJob; onDelete: (job: DraftJob) => void }) {
  const t = useT(listMessages);
  if (!isDraft(job.state)) return null;
  const label = t('moreFor', { name: jobName(job) });
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={label}
            title={label}
            icon={<DotsThreeIcon size={20} weight="bold" />}
            className="pointer-coarse:size-11"
          />
        }
      />
      <DropdownMenu.Content align="end" className="min-w-48">
        <DropdownMenu.Item variant="danger" className="gap-2" onClick={() => onDelete(job)}>
          <TrashIcon size={16} aria-hidden="true" />
          {t('deleteDraft')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

/** The draft the dialog names: the move, the admin and the old panel's host, then the new panel. */
function DraftFacts({ job }: { job: DraftJob }) {
  const host = originHost(job.source.origin);
  return (
    <div className="flex flex-col gap-2 rounded-lg bg-kumo-tint px-4 py-3">
      <span className="flex min-w-0 items-center gap-3">
        <JobMove job={job} />
        <span className="flex min-w-0 flex-col">
          <bdi dir="ltr" className="truncate text-start font-medium text-kumo-strong">
            {jobName(job)}
          </bdi>
          {job.source.admin ? (
            <bdi dir="ltr" className="truncate text-start text-sm text-kumo-subtle">
              {host}
            </bdi>
          ) : null}
        </span>
      </span>
      <PanelLabel
        kind={job.panel.kind || undefined}
        name={job.panel.name}
        className="min-w-0 text-sm"
      />
    </div>
  );
}

/**
 * Delete draft for any list of jobs: `ask` opens one confirmation that
 * names the draft; on success a toast says so and every job list reloads.
 * Render `dialog` once next to the list.
 */
export function useDeleteDraft() {
  const t = useT(listMessages);
  const td = useT(dialogMessages);
  const errorText = useErrorText();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  // The job stays after closing, so the dialog keeps its words while it fades out.
  const [job, setJob] = useState<DraftJob | null>(null);
  const [open, setOpen] = useState(false);
  const remove = useMutation({
    mutationFn: (j: DraftJob) => deleteJob(j.id),
    onSuccess: () => {
      setOpen(false);
      toasts.add({ title: t('draftDeleted') });
      void qc.invalidateQueries({ queryKey: migrationKeys.all });
    },
  });
  const ask = (j: DraftJob) => {
    remove.reset();
    setJob(j);
    setOpen(true);
  };
  const dialog = (
    <Dialog.Root open={open} onOpenChange={(next) => !remove.isPending && setOpen(next)}>
      <Dialog className="flex flex-col gap-4 p-6" size="base">
        <Dialog.Title className="text-heading font-semibold">{t('deleteTitle')}</Dialog.Title>
        {job ? <DraftFacts job={job} /> : null}
        <Dialog.Description className="text-kumo-subtle">{t('deleteBody')}</Dialog.Description>
        {remove.isError ? (
          <InlineAlert>{`${t('deleteFailed')} ${errorText(remove.error)}`}</InlineAlert>
        ) : null}
        <div className="flex flex-wrap justify-end gap-2">
          <Button
            className="pointer-coarse:h-11"
            disabled={remove.isPending}
            onClick={() => setOpen(false)}
          >
            {td('cancel')}
          </Button>
          <Button
            variant="destructive"
            className="pointer-coarse:h-11"
            loading={remove.isPending}
            onClick={() => job && remove.mutate(job)}
          >
            {t('deleteDraft')}
          </Button>
        </div>
      </Dialog>
    </Dialog.Root>
  );
  return { ask, dialog };
}
