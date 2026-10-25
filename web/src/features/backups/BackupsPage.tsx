import { RefreshButton } from '@cloudflare/kumo/components/button';
import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { ArchiveIcon, ShieldCheckIcon } from '@phosphor-icons/react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { migrationKeys } from '../../lib/api/migrations';
import type { Panel } from '../../lib/api/panels';
import { backupKeys, deleteBackup, restoreBackup } from '../../lib/api/safemode';
import { useErrorText } from '../../lib/errors/errors';
import { useFormat } from '../../lib/preferences/region';
import { useBackups } from '../overview/use-overview';
import { useNow } from '../overview/use-now';
import { controlMessages, controlText } from '../panels/control';
import { usePanels } from '../panels/use-panels';
import { backupRows, blockedPanels, type BackupRow } from './backup-rows';
import { BackupsTable } from './BackupsTable';
import { backupsMessages } from './messages';

/** A panel Karvan can't stop and start: why, and how to allow it. */
function ControlNotice({ panel }: { panel: Panel }) {
  const t = useT(backupsMessages);
  const tc = useT(controlMessages);
  if (!panel.control) return null;
  const text = controlText(panel.control);
  return (
    <InlineAlert tone="warning">
      <span className="flex flex-col gap-1">
        <span>{t('controlNotice', { name: panel.name })}</span>
        <span className="text-kumo-subtle">
          <CodeText text={tc(text.hint)} /> {text.fix ? <CodeText text={tc(text.fix)} /> : null}
        </span>
      </span>
    </InlineAlert>
  );
}

function Empty() {
  const t = useT(backupsMessages);
  const navigate = useNavigate();
  return (
    <EmptyState
      icon={<ArchiveIcon size={32} aria-hidden="true" />}
      title={t('emptyTitle')}
      action={
        <Button
          icon={<ShieldCheckIcon size={16} />}
          className="pointer-coarse:h-11"
          onClick={() => void navigate('/settings/safe-mode')}
        >
          {t('safeModeSettings')}
        </Button>
      }
    >
      {t('emptyBody')}
    </EmptyState>
  );
}

/** Put back and Delete, each behind the backup id typed (4.10). */
function useBackupActions() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const toasts = useKumoToastManager();
  const t = useT(backupsMessages);
  const [restoring, setRestoring] = useState<BackupRow | null>(null);
  const [removing, setRemoving] = useState<BackupRow | null>(null);
  const restore = useMutation({
    mutationFn: (r: BackupRow) => restoreBackup(r.backup.id, r.backup.id),
    onSuccess: (res) => {
      void qc.invalidateQueries({ queryKey: backupKeys.all });
      void qc.invalidateQueries({ queryKey: migrationKeys.all });
      void navigate(`/migrations/${encodeURIComponent(res.jobId)}/live?after=${String(res.after)}`);
    },
  });
  const remove = useMutation({
    mutationFn: (r: BackupRow) => deleteBackup(r.backup.id),
    onSuccess: () => {
      setRemoving(null);
      toasts.add({ title: t('deleted') });
      void qc.invalidateQueries({ queryKey: backupKeys.all });
    },
  });
  return { restoring, setRestoring, restore, removing, setRemoving, remove };
}

/** Safe Mode backups (4.10): which job took each, Put back and Delete. */
export function BackupsPage() {
  const t = useT(backupsMessages);
  usePageTitle(t('title'));
  const fmt = useFormat();
  const now = useNow();
  const errorText = useErrorText();
  const backups = useBackups();
  const panels = usePanels();
  const a = useBackupActions();
  const rows = backups.data ? backupRows(backups.data, panels.data ?? []) : null;
  let body;
  if (backups.isError) {
    body = (
      <ErrorState
        message={t('loadFailed')}
        error={backups.error}
        onRetry={() => void backups.refetch()}
      />
    );
  } else if (!rows || panels.isPending) {
    body = (
      <div role="status" aria-label={t('title')} className="flex flex-col gap-3">
        <Bone className="h-10 w-full rounded-lg" />
        <Bone className="h-40 w-full rounded-lg" />
      </div>
    );
  } else if (rows.length === 0) {
    body = <Empty />;
  } else {
    body = (
      <>
        {blockedPanels(rows).map((p) => (
          <ControlNotice key={p.id} panel={p} />
        ))}
        <BackupsTable
          rows={rows}
          now={now}
          actions={{
            onRestore: (r) => {
              a.restore.reset();
              a.setRestoring(r);
            },
            onDelete: (r) => {
              a.remove.reset();
              a.setRemoving(r);
            },
          }}
        />
        <p className="text-sm text-kumo-subtle">{t('shown', { count: rows.length })}</p>
      </>
    );
  }
  const r = a.restoring;
  return (
    <div className="flex flex-col">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <RefreshButton
            variant="secondary"
            aria-label={t('refresh')}
            title={t('refresh')}
            loading={backups.isFetching || panels.isFetching}
            onClick={() => {
              void backups.refetch();
              void panels.refetch();
            }}
            className="pointer-coarse:size-11"
          />
        }
      />
      <div className="flex max-w-[1400px] flex-col gap-4 p-4 sm:p-8">{body}</div>
      <TypeToConfirm
        open={r !== null}
        onOpenChange={(open) => {
          if (!open && !a.restore.isPending) a.setRestoring(null);
        }}
        title={t('restoreTitle')}
        word={r?.backup.id ?? ''}
        action={t('restore')}
        pending={a.restore.isPending || a.restore.isSuccess}
        error={
          a.restore.isError ? `${t('restoreFailed')} ${errorText(a.restore.error)}` : undefined
        }
        onConfirm={() => r && a.restore.mutate(r)}
      >
        {t('restoreBody', {
          name: r?.panel?.name ?? '',
          time: r ? fmt.dateTime(new Date(r.backup.createdAt)) : '',
        })}
      </TypeToConfirm>
      <TypeToConfirm
        open={a.removing !== null}
        onOpenChange={(open) => {
          if (!open) a.setRemoving(null);
        }}
        title={t('deleteTitle')}
        word={a.removing?.backup.id ?? ''}
        action={t('delete')}
        pending={a.remove.isPending}
        error={a.remove.isError ? `${t('deleteFailed')} ${errorText(a.remove.error)}` : undefined}
        onConfirm={() => a.removing && a.remove.mutate(a.removing)}
      >
        {t('deleteBody')}
      </TypeToConfirm>
    </div>
  );
}
