import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { PlusIcon, StackIcon } from '@phosphor-icons/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { deleteTemplate, listTemplates, templateKeys } from '../../lib/api/templates';
import { useErrorText } from '../../lib/errors/errors';
import type { StoredTemplate } from '../../lib/plan/changes';
import { useNow } from '../overview/use-now';
import { newMigrationPath } from '../wizard/steps';
import { templatesMessages } from './messages';
import { TemplateCard } from './TemplateCard';
import { Reveal } from '../../components/feedback/Reveal';
import { useFirstLoad } from '../../lib/motion/use-first-load';

/** Saved templates with what each does, and Delete behind the typed name (4.8). */
export function TemplatesPage() {
  const t = useT(templatesMessages);
  usePageTitle(t('title'));
  const navigate = useNavigate();
  const now = useNow();
  const qc = useQueryClient();
  const toasts = useKumoToastManager();
  const errorText = useErrorText();
  const list = useQuery({ queryKey: templateKeys.all, queryFn: listTemplates });
  const [asking, setAsking] = useState<StoredTemplate | null>(null);
  const remove = useMutation({
    mutationFn: (x: StoredTemplate) => deleteTemplate(x.id),
    onSuccess: () => {
      setAsking(null);
      toasts.add({ title: t('deleted') });
      void qc.invalidateQueries({ queryKey: templateKeys.all });
    },
  });
  const load = useFirstLoad(list.data);
  let body;
  if (list.isError) {
    body = (
      <ErrorState
        message={t('loadFailed')}
        error={list.error}
        onRetry={() => void list.refetch()}
      />
    );
  } else if (!load.data) {
    body = (
      <div role="status" aria-label={t('title')} className="flex flex-col gap-4">
        <Bone className="h-32 w-full rounded-lg" />
        <Bone className="h-32 w-full rounded-lg" />
      </div>
    );
  } else if (load.data.length === 0) {
    body = (
      <EmptyState
        icon={<StackIcon size={32} aria-hidden="true" />}
        title={t('emptyTitle')}
        action={
          <Button
            icon={<PlusIcon size={16} />}
            className="pointer-coarse:h-9"
            onClick={() => void navigate(newMigrationPath)}
          >
            {t('newMigration')}
          </Button>
        }
      >
        {t('emptyBody')}
      </EmptyState>
    );
  } else {
    body = (
      <section aria-label={t('list')} className="flex flex-col gap-4">
        {load.data.map((x) => (
          <TemplateCard
            key={x.id}
            template={x}
            now={now}
            onDelete={() => {
              remove.reset();
              setAsking(x);
            }}
          />
        ))}
      </section>
    );
  }
  return (
    <div className="flex flex-col">
      <PageHeader title={t('title')} description={t('description')} />
      <div className="flex max-w-[1100px] flex-col gap-4 p-4 sm:p-8">
        <Reveal on={load.reveal}>{body}</Reveal>
      </div>
      <TypeToConfirm
        open={asking !== null}
        onOpenChange={(open) => {
          if (!open) setAsking(null);
        }}
        title={t('deleteTitle')}
        word={asking?.template.name ?? ''}
        action={t('delete')}
        pending={remove.isPending}
        error={remove.isError ? `${t('deleteFailed')} ${errorText(remove.error)}` : undefined}
        onConfirm={() => asking && remove.mutate(asking)}
      >
        {t('deleteBody')}
      </TypeToConfirm>
    </div>
  );
}
