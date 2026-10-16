import { useQuery } from '@tanstack/react-query';
import { getSession, signOut } from '../../lib/api/session';

export const sessionKey = ['session'] as const;

/** The signed-in operator; also refreshes the CSRF token the client sends. */
export function useSession() {
  return useQuery({ queryKey: sessionKey, queryFn: getSession, staleTime: Infinity });
}

/** The server serves the sign-in page at the same address once the session is gone. */
export function returnToSignIn() {
  window.location.replace(document.baseURI);
}

export function useSignOut() {
  return () => {
    void signOut()
      .catch(() => undefined)
      .finally(returnToSignIn);
  };
}
