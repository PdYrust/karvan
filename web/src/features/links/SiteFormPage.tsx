import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { LinkSimpleIcon } from '@phosphor-icons/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import {
  getLinkSites,
  linkKeys,
  saveSite,
  sitePrefill,
  type LinkSite,
  type SiteInput,
} from '../../lib/api/links';
import { useCurrentPanel } from '../panels/current-panel';
import { siteFormMessages } from './form-messages';
import { linksMessages } from './messages';
import { newValues, valuesOf } from './site-form';
import { SiteForm } from './SiteForm';

/** Adds a site (`/links/new`, maybe prefilled from a migration) or changes one (`/links/:id`). */
export function SiteFormPage() {
  const t = useT(siteFormMessages);
  const tl = useT(linksMessages);
  const { id } = useParams();
  const siteId = id ? Number(id) : undefined;
  const [query] = useSearchParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const toasts = useKumoToastManager();
  const { panel } = useCurrentPanel();
  const list = useQuery({ queryKey: linkKeys.all, queryFn: getLinkSites, enabled: !!siteId });
  const site: LinkSite | undefined = list.data?.sites.find((s) => s.id === siteId);
  usePageTitle(site ? t('editTitle', { name: site.name }) : t('newTitle'));
  const save = useMutation({
    mutationFn: (input: SiteInput) => saveSite(input, siteId),
    onSuccess: () => {
      toasts.add({ title: siteId ? t('saved') : t('added') });
      void qc.invalidateQueries({ queryKey: linkKeys.all });
      void navigate('/links');
    },
  });
  const back = () => void navigate('/links');
  let body;
  if (siteId && list.isError) {
    body = (
      <ErrorState
        message={tl('loadFailed')}
        error={list.error}
        onRetry={() => void list.refetch()}
      />
    );
  } else if (siteId && !list.data) {
    body = (
      <div role="status" aria-label={t('editTitle', { name: '' })} className="flex flex-col gap-4">
        <Bone className="h-10 w-full max-w-[720px]" />
        <Bone className="h-40 w-full max-w-[720px]" />
      </div>
    );
  } else if (siteId && !site) {
    body = (
      <EmptyState
        icon={<LinkSimpleIcon size={32} aria-hidden="true" />}
        title={t('missing')}
        action={<Button onClick={back}>{tl('title')}</Button>}
      />
    );
  } else {
    body = (
      <SiteForm
        key={site?.id ?? 'new'}
        initial={site ? valuesOf(site) : newValues(sitePrefill(query), panel?.id ?? 0)}
        siteId={siteId}
        hasKey={site?.hasKey ?? false}
        saving={save.isPending}
        error={save.error}
        onSave={(input) => save.mutate(input)}
        onCancel={back}
      />
    );
  }
  return (
    <div className="flex flex-col">
      <PageHeader
        eyebrow={
          <Link
            to="/links"
            className="inline-flex items-center hover:text-kumo-default pointer-coarse:min-h-11"
          >
            {tl('title')}
          </Link>
        }
        title={
          site ? (
            <>
              {t('editTitle', { name: '' })}
              <bdi>{site.name}</bdi>
            </>
          ) : (
            t('newTitle')
          )
        }
        description={t('description')}
      />
      <div className="p-4 sm:p-8">{body}</div>
    </div>
  );
}
