import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT, type Messages } from '../../i18n/locale';
import { getJob, migrationKeys } from '../../lib/api/migrations';
import { JobStateLabel } from '../migrations/JobLabels';
const messages: Messages<'eyebrow' | 'title' | 'soon'> = {
  en: { eyebrow: 'Migrations', title: 'Live progress', soon: 'The live view opens here soon.' },
  fa: { eyebrow: 'انتقال ها', title: 'پیشرفت زنده', soon: 'نمای زنده به زودی همینجا باز میشود.' },
};
/** Where Start leads (4.6). It holds the place of the live view, with the job's state. */
export function LivePage() {
  const { id = '' } = useParams();
  const t = useT(messages);
  usePageTitle(t('title'));
  const job = useQuery({ queryKey: migrationKeys.job(id), queryFn: () => getJob(id) });
  return (
    <div className="flex flex-col">
      <PageHeader eyebrow={t('eyebrow')} title={t('title')} />
      <div className="flex max-w-[720px] flex-col gap-4 p-4 sm:p-8">
        {job.data ? <JobStateLabel state={job.data.state} /> : null}
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('soon')}</p>
      </div>
    </div>
  );
}
