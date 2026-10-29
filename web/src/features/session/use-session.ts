import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useQuery } from '@tanstack/react-query';
import { useT } from '../../i18n/locale';
import { ApiError } from '../../lib/api/client';
import { getSession, signOut } from '../../lib/api/session';
import { returnTarget } from '../../lib/return-target';
import { sessionMessages } from './messages';

export const sessionKey = ['session'] as const;

/** The signed-in operator; also refreshes the CSRF token the client sends. */
export function useSession() {
  return useQuery({ queryKey: sessionKey, queryFn: getSession, staleTime: Infinity });
}

/** Reloads the current page; without a session the server serves the sign-in page there. */
export function returnToSignIn() {
  window.location.replace(returnTarget(window.location, document.baseURI));
}

/** Signs out; on a failure the page stays, so a shared machine is never left signed in by surprise. */
export function useSignOut() {
  const t = useT(sessionMessages);
  const toasts = useKumoToastManager();
  return () => {
    signOut().then(returnToSignIn, (err: unknown) => {
      if (err instanceof ApiError && err.status === 401) returnToSignIn();
      else toasts.add({ title: t('signOutFailed'), variant: 'error' });
    });
  };
}
