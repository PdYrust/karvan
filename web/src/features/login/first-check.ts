import { getSession } from '../../lib/api/session';
import { classify, type SignInError } from './use-sign-in';

/** What the session says before the sign-in page paints anything. */
export interface FirstCheck {
  signedIn: boolean;
  /** Why the check failed, shown on the form; null when it answered. */
  error: SignInError | null;
}

/**
 * Reads the session once, which also fetches the token that guards the
 * form. The server sends the sign-in page whenever the page's own request
 * came without the session cookie, as it does when the installed app
 * opens, so a live session can still be there.
 */
export async function firstCheck(): Promise<FirstCheck> {
  try {
    const s = await getSession();
    return { signedIn: s.authenticated, error: null };
  } catch (e) {
    return { signedIn: false, error: classify(e) };
  }
}
