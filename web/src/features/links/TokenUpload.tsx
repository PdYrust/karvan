import { Button } from '@cloudflare/kumo/components/button';
import { UploadSimpleIcon } from '@phosphor-icons/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRef } from 'react';
import { Bar } from '../../components/charts/Bar';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import { getUpload, linkKeys, uploadTokens, type UploadProgress } from '../../lib/api/links';
import { useErrorText } from '../../lib/errors/errors';
import { linksMessages, type LinksKey } from './messages';

const outcomes = ['stored', 'known', 'noUser', 'refused', 'failed'] as const;

/** The progress of a site's token list, read every second while it runs. */
function useUpload(siteId: number) {
  const qc = useQueryClient();
  const key = linkKeys.upload(siteId);
  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const before = qc.getQueryData<UploadProgress>(key);
      const now = await getUpload(siteId);
      // The site's numbers changed once a list ends.
      if (before?.state === 'running' && now.state !== 'running') {
        void qc.invalidateQueries({ queryKey: linkKeys.all, exact: true });
      }
      return now;
    },
    refetchInterval: (q) => (q.state.data?.state === 'running' ? 1000 : false),
  });
}

function Progress({ p }: { p: UploadProgress }) {
  const t = useT(linksMessages);
  if (p.state === 'running') {
    return (
      <div className="flex flex-col gap-2">
        <p className="text-sm text-kumo-default tabular">
          {t('checking', { checked: p.checked, found: p.found })}
        </p>
        <Bar share={p.found ? p.checked / p.found : 0} />
      </div>
    );
  }
  if (p.found === 0) return <p className="text-sm text-kumo-subtle">{t('upNone')}</p>;
  return (
    <div className="flex flex-col gap-1 text-sm">
      <p className={p.state === 'stopped' ? 'text-kumo-default' : 'text-kumo-strong'}>
        {p.state === 'stopped' ? t('uploadStopped') : t('checkedAll', { count: p.found })}
      </p>
      <ul className="flex flex-col gap-0.5 text-kumo-subtle tabular">
        {outcomes
          .filter((o) => p[o] > 0)
          .map((o) => (
            <li key={o}>{t(`up_${o}` as LinksKey, { count: p[o] })}</li>
          ))}
      </ul>
    </div>
  );
}

/** Upload a file of old links; the checks run on the server and their counter shows here. */
export function TokenUpload({ siteId }: { siteId: number }) {
  const t = useT(linksMessages);
  const errorText = useErrorText();
  const qc = useQueryClient();
  const input = useRef<HTMLInputElement>(null);
  const progress = useUpload(siteId);
  const send = useMutation({
    mutationFn: (file: File) => uploadTokens(siteId, file),
    onSuccess: (p) => qc.setQueryData(linkKeys.upload(siteId), p),
  });
  const p = progress.data;
  const running = p?.state === 'running' || send.isPending;
  return (
    <div className="flex flex-col items-start gap-3">
      <p className="text-sm text-kumo-subtle">{t('uploadHint')}</p>
      <div aria-live="polite" className="w-full empty:hidden">
        {p && p.state !== 'none' ? <Progress p={p} /> : null}
      </div>
      {send.isError ? (
        <InlineAlert className="w-full">
          {`${t('uploadFailed')} ${errorText(send.error)}`}
        </InlineAlert>
      ) : null}
      <input
        ref={input}
        type="file"
        accept=".txt,.csv,text/plain,text/csv"
        className="sr-only"
        tabIndex={-1}
        aria-hidden="true"
        onChange={(e) => {
          const file = e.target.files?.[0];
          e.target.value = '';
          if (file) send.mutate(file);
        }}
      />
      <Button
        icon={<UploadSimpleIcon size={16} />}
        loading={send.isPending}
        disabled={running}
        className="pointer-coarse:h-9"
        onClick={() => input.current?.click()}
      >
        {p && p.state !== 'none' ? t('uploadAgain') : t('upload')}
      </Button>
    </div>
  );
}
