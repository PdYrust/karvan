import { useQuery } from '@tanstack/react-query';
import { getOverview } from '../../lib/api/overview';

export const overviewKey = ['overview'] as const;

export function useOverview() {
  return useQuery({ queryKey: overviewKey, queryFn: getOverview });
}
