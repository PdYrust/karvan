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
import { useLocale, useT } from '../../i18n/locale';
import { ListToolbar } from '../../components/data/ListToolbar';
import { dataMessages } from '../../components/data/messages';
import { sortChoice } from '../../components/data/sort-choice';
import { matchesTerm, sortRows, type ListSort } from '../../lib/lists/choices';
import { deleteSite, getLinkSites, linkKeys, type LinkSite } from '../../lib/api/rules';
import { useErrorText } from '../../lib/errors/errors';
import { useFormat } from '../../lib/preferences/region';
import { usePanels } from '../panels/use-panels';
import { rulesMessages } from './messages';
import { SiteCard } from './SiteCard';
import { Reveal } from '../../components/feedback/Reveal';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { useLingering } from '../../lib/motion/use-lingering';

function AddButton() {
  const t = useT(rulesMessages);
  const navigate = useNavigate();
  return (
    <Button
      variant="primary"
      icon={<PlusIcon size={16} />}
      className="pointer-coarse:h-9"
      onClick={() => void navigate('/rules/new')}
    >
      {t('add')}
    </Button>
  );
}

/** One card per old link site, with ports that could not open first (4.9). */
export function RulesPage() {
  const t = useT(rulesMessages);
  usePageTitle(t('title'));
  const fmt = useFormat();
  const qc = useQueryClient();
  const toasts = useKumoToastManager();
  const errorText = useErrorText();
  const list = useQuery({ queryKey: linkKeys.all, queryFn: getLinkSites });
  const panels = usePanels();
  const [asking, setAsking] = useState<LinkSite | null>(null);
  const td = useT(dataMessages);
  const { locale } = useLocale();
  const [query, setQuery] = useState('');
  const [tls, setTls] = useState<'all' | 'https' | 'plain'>('all');
  const [sort, setSort] = useState<ListSort>('name');
  // The dialog keeps the name while it fades out.
  const named = useLingering(asking);
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
    const sites = load.data.sites;
    const isTls = (s: LinkSite, f: typeof tls) => f === 'all' || (f === 'https') === s.https;
    const shown = sortRows(
      sites.filter(
        (s) => isTls(s, tls) && matchesTerm(query, [s.name, s.sourceUrl, ...(s.domains ?? [])]),
      ),
      sort,
      { name: (s) => s.name, time: (s) => s.id },
      locale,
    );
    const tlsText = { all: td('showAll'), https: t('https'), plain: t('plainHttp') };
    body = (
      <>
        {ports.length > 0 ? (
          <InlineAlert tone="warning">{t('portsFailed', { ports: ports.join(', ') })}</InlineAlert>
        ) : null}
        <ListToolbar
          search={{ label: t('search'), value: query, onChange: setQuery }}
          choices={[
            {
              label: td('filter'),
              value: tls,
              all: 'all',
              options: (['all', 'https', 'plain'] as const).map((f) => ({
                value: f,
                label: tlsText[f],
                text: tlsText[f],
                count: sites.filter((s) => isTls(s, f)).length,
              })),
              onChange: (f) => setTls(f as typeof tls),
            },
            sortChoice(td, sort, setSort, ['name', 'newest']),
          ]}
        />
        {shown.length === 0 ? (
          <p className="rounded-lg bg-kumo-base px-4 py-8 text-center text-kumo-subtle ring ring-kumo-hairline">
            {t('noMatch')}
          </p>
        ) : null}
        <section aria-label={t('list')} className="flex flex-col gap-4">
          {shown.map((s) => (
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
        word={named?.name ?? ''}
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
