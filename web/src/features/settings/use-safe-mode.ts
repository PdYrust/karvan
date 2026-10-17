import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { getSafeMode, putSafeMode, type SafeMode } from '../../lib/api/safemode';

export const safeModeKey = ['settings', 'safe-mode'] as const;

export function useSafeMode() {
  return useQuery({ queryKey: safeModeKey, queryFn: getSafeMode, staleTime: 30_000 });
}

export function useSetSafeMode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ next, confirm }: { next: SafeMode; confirm?: string }) =>
      putSafeMode(next, confirm),
    onSuccess: (saved) => queryClient.setQueryData(safeModeKey, saved),
  });
}
