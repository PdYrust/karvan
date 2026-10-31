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
import { ListSkeleton } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useLocale, useT } from '../../i18n/locale';
import { ListToolbar } from '../../components/data/ListToolbar';
import { ToolbarAdd } from '../../components/data/ToolbarAdd';
import { dataMessages } from '../../components/data/messages';
import { sortChoice } from '../../components/data/sort-choice';
import { matchesTerm, sortRows, type ListSort } from '../../lib/lists/choices';
import { domainKeys, getDomains } from '../../lib/api/domains';
import {
  deleteRedirect,
  deleteSite,
  getLinkSites,
  getRedirects,
  linkKeys,
  redirectKeys,
  type LinkSite,
  type Redirect,
} from '../../lib/api/rules';
import { useErrorText } from '../../lib/errors/errors';
import { useFormat } from '../../lib/preferences/region';
import { usePanels } from '../panels/use-panels';
import { rulesMessages } from './messages';
import { RedirectCard } from './RedirectCard';
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

/** A panel rule or a redirect, as the list holds them. */
type Rule = { kind: 'panel'; rule: LinkSite } | { kind: 'redirect'; rule: Redirect };
const terms = (r: Rule) =>
  r.kind === 'panel'
    ? [r.rule.name, r.rule.sourceUrl, ...(r.rule.domains ?? [])]
    : [r.rule.name, r.rule.toUrl, ...(r.rule.domains ?? [])];

/** One card per rule, panel rules and redirects, with ports that could not open first. */
export function RulesPage() {
  const navigate = useNavigate();
  const t = useT(rulesMessages);
  usePageTitle(t('title'));
  const fmt = useFormat();
  const qc = useQueryClient();
  const toasts = useKumoToastManager();
  const errorText = useErrorText();
  const list = useQuery({ queryKey: linkKeys.all, queryFn: getLinkSites });
  const redirects = useQuery({ queryKey: redirectKeys.all, queryFn: getRedirects });
  const domains = useQuery({ queryKey: domainKeys.all, queryFn: getDomains });
  const panels = usePanels();
  const [asking, setAsking] = useState<Rule | null>(null);
  const td = useT(dataMessages);
  const { locale } = useLocale();
  const [query, setQuery] = useState('');
  const [tls, setTls] = useState<'all' | 'https' | 'plain'>('all');
  const [sort, setSort] = useState<ListSort>('name');
  // The dialog keeps the name while it fades out.
  const named = useLingering(asking);
  const remove = useMutation({
    mutationFn: (r: Rule) =>
      r.kind === 'panel' ? deleteSite(r.rule.id) : deleteRedirect(r.rule.id),
    onSuccess: (_, r) => {
      setAsking(null);
      toasts.add({ title: t('deleted') });
      // A deleted site has no upload to read any more.
      if (r.kind === 'panel') qc.removeQueries({ queryKey: linkKeys.upload(r.rule.id) });
      void qc.invalidateQueries({
        queryKey: r.kind === 'panel' ? linkKeys.all : redirectKeys.all,
        exact: true,
      });
      void qc.invalidateQueries({ queryKey: domainKeys.all });
    },
  });
  const both =
    list.data && redirects.data && domains.data
      ? { ...list.data, redirects: redirects.data.redirects, domains: domains.data.domains }
      : undefined;
  const load = useFirstLoad(both);
  const failed = [list, redirects, domains].find((q) => q.isError);
  let body;
  if (failed) {
    body = (
      <ErrorState
        message={t('loadFailed')}
        error={failed.error}
        onRetry={() => void failed.refetch()}
      />
    );
  } else if (!load.data) {
    body = <ListSkeleton label={t('title')} />;
  } else if (load.data.sites.length + load.data.redirects.length === 0) {
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
    const rules: Rule[] = [
      ...load.data.sites.map((rule) => ({ kind: 'panel' as const, rule })),
      ...load.data.redirects.map((rule) => ({ kind: 'redirect' as const, rule })),
    ];
    const isTls = (r: Rule, f: typeof tls) => f === 'all' || (f === 'https') === r.rule.https;
    const shown = sortRows(
      rules.filter((r) => isTls(r, tls) && matchesTerm(query, terms(r))),
      sort,
      { name: (r) => r.rule.name, time: (r) => r.rule.id },
      locale,
    );
    const known = load.data.domains;
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
                count: rules.filter((r) => isTls(r, f)).length,
              })),
              onChange: (f) => setTls(f as typeof tls),
            },
            sortChoice(td, sort, setSort, ['name', 'newest']),
          ]}
        >
          <ToolbarAdd label={t('add')} onClick={() => void navigate('/rules/new')} />
        </ListToolbar>
        {shown.length === 0 ? (
          <p className="rounded-lg bg-kumo-base px-4 py-8 text-center text-kumo-subtle ring ring-kumo-hairline">
            {t('noMatch')}
          </p>
        ) : null}
        <section aria-label={t('list')} className="flex flex-col gap-4">
          {shown.map((r) => {
            const onDelete = () => {
              remove.reset();
              setAsking(r);
            };
            return r.kind === 'panel' ? (
              <SiteCard
                key={`p${String(r.rule.id)}`}
                site={r.rule}
                panel={panels.data?.find((p) => p.id === r.rule.panelId)}
                domains={known}
                onDelete={onDelete}
              />
            ) : (
              <RedirectCard
                key={`r${String(r.rule.id)}`}
                redirect={r.rule}
                domains={known}
                onDelete={onDelete}
              />
            );
          })}
        </section>
      </>
    );
  }
  return (
    <div className="flex flex-col">
      <PageHeader title={t('title')} description={t('description')} />
      <div className="flex flex-col gap-4 p-4 sm:p-8">
        <Reveal on={load.reveal}>{body}</Reveal>
      </div>
      <TypeToConfirm
        open={asking !== null}
        onOpenChange={(open) => {
          if (!open) setAsking(null);
        }}
        title={t('deleteTitle')}
        word={named?.rule.name ?? ''}
        action={t('delete')}
        pending={remove.isPending}
        error={remove.isError ? `${t('deleteFailed')} ${errorText(remove.error)}` : undefined}
        onConfirm={() => asking && remove.mutate(asking)}
      >
        {named?.kind === 'redirect' ? t('deleteRedirectBody') : t('deleteBody')}
      </TypeToConfirm>
    </div>
  );
}
