import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { DownloadSimpleIcon } from '@phosphor-icons/react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Bone } from '../../components/feedback/PageSkeleton';
import { useT } from '../../i18n/locale';
import { originOf, type LinkSite } from '../../lib/api/links';
import { jobOutcome, listJobs, migrationKeys, originHost } from '../../lib/api/migrations';
import { reportFiles } from '../../lib/api/report';
import { useErrorText } from '../../lib/errors/errors';
import { useFormat } from '../../lib/preferences/region';
import { linksMessages } from './messages';

/** Done migrations into the site's panel, the latest 100: the ones from its old panel have its new links. */
function useSiteJobs(site: LinkSite) {
  return useQuery({
    queryKey: [...migrationKeys.all, 'links', site.panelId],
    queryFn: () => listJobs({ panel: site.panelId, states: ['done'], limit: 100 }),
    select: (page) =>
      page.jobs.filter(
        (j) =>
          originOf(j.source.origin) === originOf(site.sourceUrl) && (jobOutcome(j)?.moved ?? 0) > 0,
      ),
  });
}

/** Each done migration from this old panel, with its new links file. */
export function NewLinks({ site }: { site: LinkSite }) {
  const t = useT(linksMessages);
  const fmt = useFormat();
  const errorText = useErrorText();
  const toasts = useKumoToastManager();
  const jobs = useSiteJobs(site);
  const file = useMutation({
    mutationFn: (id: string) => reportFiles.links(id),
    onError: (err) =>
      toasts.add({ title: t('downloadFailed'), description: errorText(err), variant: 'error' }),
  });
  if (jobs.isError) return <p className="text-sm text-kumo-subtle">{t('newLinksFailed')}</p>;
  if (!jobs.data) return <Bone className="h-10 w-full" />;
  if (jobs.data.length === 0) return <p className="text-sm text-kumo-subtle">{t('noNewLinks')}</p>;
  return (
    <ul className="flex flex-col">
      {jobs.data.map((j) => {
        const admin = j.source.admin || originHost(j.source.origin);
        const ended = j.endedAt ? new Date(j.endedAt) : undefined;
        return (
          <li
            key={j.id}
            className="flex items-center gap-3 border-b border-kumo-hairline py-1.5 last:border-0"
          >
            <span className="flex min-w-0 flex-1 flex-col">
              <bdi className="truncate font-medium text-kumo-strong">{admin}</bdi>
              {ended ? (
                <time dateTime={ended.toISOString()} className="text-sm text-kumo-subtle">
                  {fmt.dateTime(ended)}
                </time>
              ) : null}
            </span>
            <Button
              variant="ghost"
              size="sm"
              icon={<DownloadSimpleIcon size={16} />}
              aria-label={t('downloadOf', { admin, date: ended ? fmt.dateTime(ended) : '' })}
              loading={file.isPending && file.variables === j.id}
              className="pointer-coarse:h-11"
              onClick={() => file.mutate(j.id)}
            >
              {t('download')}
            </Button>
          </li>
        );
      })}
    </ul>
  );
}
