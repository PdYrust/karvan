import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { LinkSimpleIcon } from '@phosphor-icons/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { batchKeys, listBatches, type Batch } from '../../lib/api/batches';
import { domainKeys, getDomains } from '../../lib/api/domains';
import {
  getLinkSites,
  getRedirects,
  linkKeys,
  redirectKeys,
  saveRedirect,
  saveSite,
} from '../../lib/api/rules';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { useCurrentPanel } from '../panels/current-panel';
import { siteFormMessages } from './form-messages';
import { rulesMessages } from './messages';
import {
  newRuleValues,
  redirectInputOf,
  rulePrefill,
  siteInputOf,
  valuesOfRedirect,
  valuesOfSite,
  type RuleValues,
} from './rule-form';
import { RuleForm } from './RuleForm';

const activeBatches = { state: 'active', limit: 100 } as const;

/** Reads what the form needs at once: the domains, the batches and, on a change, the rule. */
function useRuleData(id: number | undefined, redirect: boolean) {
  const domains = useQuery({ queryKey: domainKeys.all, queryFn: getDomains });
  const batches = useQuery({
    queryKey: batchKeys.list(activeBatches),
    queryFn: () => listBatches(activeBatches),
  });
  const sites = useQuery({
    queryKey: linkKeys.all,
    queryFn: getLinkSites,
    enabled: !!id && !redirect,
  });
  const redirects = useQuery({
    queryKey: redirectKeys.all,
    queryFn: getRedirects,
    enabled: !!id && redirect,
  });
  const rule = redirect ? redirects : sites;
  const failed = [domains, batches, ...(id ? [rule] : [])].find((q) => q.isError);
  let initial: RuleValues | null | undefined;
  if (id && redirect && redirects.data) {
    const r = redirects.data.redirects.find((x) => x.id === id);
    initial = r ? valuesOfRedirect(r) : null;
  } else if (id && !redirect && sites.data) {
    const s = sites.data.sites.find((x) => x.id === id);
    initial = s ? valuesOfSite(s) : null;
  }
  const ready = domains.data && batches.data && (!id || initial !== undefined);
  return {
    failed,
    ready: ready ? { domains: domains.data.domains, batches: batches.data.batches, initial } : null,
  };
}

/**
 * Add a rule (`/rules/new`, maybe from a batch: `?batch=12`) or change one
 * (`/rules/:id`, `/rules/redirects/:id`).
 */
export function RuleFormPage({ redirect = false }: { redirect?: boolean }) {
  const t = useT(siteFormMessages);
  const tl = useT(rulesMessages);
  const { id } = useParams();
  const ruleId = id ? Number(id) : undefined;
  const [query] = useSearchParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const toasts = useKumoToastManager();
  const { panel } = useCurrentPanel();
  const data = useRuleData(ruleId, redirect);
  const load = useFirstLoad(data.ready);
  const name = load.data?.initial?.name;
  usePageTitle(name ? t('editTitle', { name }) : t('newTitle'));
  const save = useMutation({
    mutationFn: ({ v, picked }: { v: RuleValues; picked: Batch[] }): Promise<unknown> =>
      v.kind === 'redirect' || v.kind === 'link'
        ? saveRedirect(redirectInputOf(v), ruleId)
        : saveSite(siteInputOf(v, picked), ruleId),
    onSuccess: () => {
      toasts.add({ title: ruleId ? t('saved') : t('added') });
      void qc.invalidateQueries({ queryKey: linkKeys.all });
      void qc.invalidateQueries({ queryKey: redirectKeys.all });
      // A domain lists the rules that hold it.
      void qc.invalidateQueries({ queryKey: domainKeys.all });
      void navigate('/rules');
    },
  });
  const back = () => void navigate('/rules');
  let body;
  if (data.failed) {
    body = (
      <ErrorState
        message={tl('loadFailed')}
        error={data.failed.error}
        onRetry={() => void data.failed?.refetch()}
      />
    );
  } else if (!load.data) {
    body = (
      <div role="status" aria-label={t('newTitle')} className="flex flex-col gap-4">
        <Bone className="h-10 w-full max-w-[720px]" />
        <Bone className="h-40 w-full max-w-[720px]" />
      </div>
    );
  } else if (load.data.initial === null) {
    body = (
      <EmptyState
        icon={<LinkSimpleIcon size={32} aria-hidden="true" />}
        title={t('missing')}
        action={<Button onClick={back}>{tl('title')}</Button>}
      />
    );
  } else {
    body = (
      <RuleForm
        key={ruleId ?? 'new'}
        initial={load.data.initial ?? newRuleValues(rulePrefill(query), panel?.id ?? 0)}
        isNew={!ruleId}
        domains={load.data.domains}
        batches={load.data.batches}
        saving={save.isPending}
        error={save.error}
        onSave={(v, picked) => save.mutate({ v, picked })}
        onCancel={back}
      />
    );
  }
  return (
    <div className="flex flex-col">
      <PageHeader
        eyebrow={
          <Link
            to="/rules"
            className="inline-flex items-center hover:text-kumo-default pointer-coarse:min-h-9"
          >
            {tl('title')}
          </Link>
        }
        title={
          name ? (
            <>
              {t('editTitle', { name: '' })}
              <bdi>{name}</bdi>
            </>
          ) : (
            t('newTitle')
          )
        }
        description={t('description')}
      />
      <div className="p-4 sm:p-8">
        <Reveal on={load.reveal}>{body}</Reveal>
      </div>
    </div>
  );
}
