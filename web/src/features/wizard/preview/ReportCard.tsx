import { Button, buttonVariants } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { DownloadSimpleIcon, PrinterIcon } from '@phosphor-icons/react';
import { useMutation } from '@tanstack/react-query';
import { useId } from 'react';
import { Link } from 'react-router';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { useT } from '../../../i18n/locale';
import { checkReportFiles } from '../../../lib/api/preflight';
import { useErrorText } from '../../../lib/errors/errors';
import { previewMessages } from './messages';
/** The pre-flight report: a printable page in a new tab, and the same as JSON or CSV. */
export function ReportCard({ jobId }: { jobId: string }) {
  const t = useT(previewMessages);
  const errorText = useErrorText();
  const titleId = useId();
  const file = useMutation({
    mutationFn: (kind: 'json' | 'csv') => checkReportFiles[kind](jobId),
  });
  const busy = file.isPending ? file.variables : undefined;
  return (
    <section
      aria-labelledby={titleId}
      className="flex flex-col gap-3 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-5"
    >
      <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
        {t('reportTitle')}
      </h2>
      <p className="max-w-[720px] text-kumo-subtle">{t('reportBody')}</p>
      {file.isError ? (
        <InlineAlert>{`${t('downloadFailed')} ${errorText(file.error)}`}</InlineAlert>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <Link
          to={`/migrations/${encodeURIComponent(jobId)}/preview/print`}
          target="_blank"
          rel="noopener"
          className={cn(buttonVariants({ variant: 'secondary' }), 'gap-2 pointer-coarse:h-9')}
        >
          <PrinterIcon size={16} aria-hidden="true" />
          {t('reportOpen')}
          <span className="sr-only">{t('newTab')}</span>
        </Link>
        {(['json', 'csv'] as const).map((kind) => (
          <Button
            key={kind}
            icon={<DownloadSimpleIcon size={16} />}
            loading={busy === kind}
            disabled={busy !== undefined}
            className="pointer-coarse:h-9"
            onClick={() => file.mutate(kind)}
          >
            {t(kind === 'json' ? 'reportJson' : 'reportCsv')}
          </Button>
        ))}
      </div>
    </section>
  );
}
