import { useQuery } from '@tanstack/react-query';
import { Navigate, useParams } from 'react-router';
import { ErrorState } from '../../components/feedback/ErrorState';
import { PageSkeleton } from '../../components/feedback/PageSkeleton';
import { useT } from '../../i18n/locale';
import { getJobRow, migrationKeys } from '../../lib/api/migrations';
import { NotFoundPage } from '../not-found/NotFoundPage';
import { jobPlace } from './job-place';
import { listMessages } from './list-messages';

/** `/migrations/:id` reads the job's row and opens what its state needs. */
export function JobRedirect() {
  const { id = '' } = useParams();
  const t = useT(listMessages);
  const row = useQuery({
    queryKey: migrationKeys.row(id),
    queryFn: () => getJobRow(id),
    // The state decides the page, so an old answer could open the wrong one.
    staleTime: 0,
  });
  if (row.isError) {
    return (
      <div className="p-4 sm:p-8">
        <ErrorState message={t('jobFailed')} error={row.error} onRetry={() => void row.refetch()} />
      </div>
    );
  }
  if (row.isPending || row.isFetching) return <PageSkeleton label={t('title')} />;
  if (!row.data) return <NotFoundPage />;
  return <Navigate to={jobPlace(row.data)} replace />;
}
