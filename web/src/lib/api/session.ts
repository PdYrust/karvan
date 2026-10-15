import { api, setCsrfToken } from './client';

export interface Operator {
  username: string;
  totpEnabled: boolean;
}

export interface SessionState {
  authenticated: boolean;
  operator?: Operator;
  csrfToken: string;
}

/** Reads the session and keeps its CSRF token for later requests. */
export async function getSession(): Promise<SessionState> {
  const s = await api<SessionState>('GET', 'session');
  setCsrfToken(s.csrfToken);
  return s;
}

export interface SignInInput {
  username: string;
  password: string;
  totpCode?: string;
}

export async function signIn(input: SignInInput): Promise<SessionState> {
  const s = await api<SessionState>('POST', 'session', {
    username: input.username,
    password: input.password,
    totpCode: input.totpCode ?? '',
  });
  setCsrfToken(s.csrfToken);
  return s;
}

export async function signOut(): Promise<void> {
  await api<undefined>('DELETE', 'session');
}
