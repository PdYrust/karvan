import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { LinkSimpleIcon, PlusIcon } from '@phosphor-icons/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { deleteSite, getLinkSites, linkKeys, type LinkSite } from '../../lib/api/links';
import { useErrorText } from '../../lib/errors/errors';
import { useFormat } from '../../lib/preferences/region';
import { usePanels } from '../panels/use-panels';
import { linksMessages } from './messages';
import { SiteCard } from './SiteCard';
import { Reveal } from '../../components/feedback/Reveal';
import { useFirstLoad } from '../../lib/motion/use-first-load';

function AddButton() {
  const t = useT(linksMessages);
  const navigate = useNavigate();
  return (
    <Button
      variant="primary"
      icon={<PlusIcon size={16} />}
      className="pointer-coarse:h-9"
      onClick={() => void navigate('/links/new')}
    >
      {t('add')}
    </Button>
  );
}

/** One card per old link site, with ports that could not open first (4.9). */
export function LinksPage() {
  const t = useT(linksMessages);
  usePageTitle(t('title'));
  const fmt = useFormat();
  const qc = useQueryClient();
  const toasts = useKumoToastManager();
  const errorText = useErrorText();
  const list = useQuery({ queryKey: linkKeys.all, queryFn: getLinkSites });
  const panels = usePanels();
  const [asking, setAsking] = useState<LinkSite | null>(null);
  const remove = useMutation({
    mutationFn: (s: LinkSite) => deleteSite(s.id),
    onSuccess: (_, s) => {
      setAsking(null);
      toasts.add({ title: t('deleted') });
      // A deleted site has no upload to read any more.
      qc.removeQueries({ queryKey: linkKeys.upload(s.id) });
      void qc.invalidateQueries({ queryKey: linkKeys.all, exact: true });
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
        <Bone className="h-72 w-full rounded-lg" />
      </div>
    );
  } else if (load.data.sites.length === 0) {
    body = (
      <EmptyState
        icon={<LinkSimpleIcon size={32} aria-hidden="true" />}
        title={t('emptyTitle')}
        action={<AddButton />}
      >
        {t('emptyBody')}
      </EmptyState>
    );
  } else {
    const ports = Object.keys(load.data.portErrors).map((p) => fmt.number(Number(p)));
    body = (
      <>
        {ports.length > 0 ? (
          <InlineAlert tone="warning">{t('portsFailed', { ports: ports.join(', ') })}</InlineAlert>
        ) : null}
        <section aria-label={t('list')} className="flex flex-col gap-4">
          {load.data.sites.map((s) => (
            <SiteCard
              key={s.id}
              site={s}
              panel={panels.data?.find((p) => p.id === s.panelId)}
              onDelete={() => {
                remove.reset();
                setAsking(s);
              }}
            />
          ))}
        </section>
      </>
    );
  }
  const hasSites = Boolean(load.data?.sites.length);
  return (
    <div className="flex flex-col">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={hasSites ? <AddButton /> : undefined}
      />
      <div className="flex max-w-[1200px] flex-col gap-4 p-4 sm:p-8">
        <Reveal on={load.reveal}>{body}</Reveal>
      </div>
      <TypeToConfirm
        open={asking !== null}
        onOpenChange={(open) => {
          if (!open) setAsking(null);
        }}
        title={t('deleteTitle')}
        word={asking?.name ?? ''}
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
