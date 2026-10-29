import { useEffect, useState } from 'react';
import { ApiError, offline } from '../../lib/api/client';
import { getSession, signIn } from '../../lib/api/session';
import { returnTarget } from '../../lib/return-target';

export type Step = 'credentials' | 'totp';

export type SignInError =
  'required' | 'invalid' | 'totpInvalid' | 'locked' | 'offline' | 'stale' | 'server';

interface Fields {
  username: string;
  password: string;
  code: string;
}

/** Goes back to the page the operator asked for; the server serves the app there once signed in. */
function enterApp() {
  window.location.replace(returnTarget(window.location, document.baseURI));
}

function classify(err: unknown): SignInError {
  if (!(err instanceof ApiError)) return 'server';
  switch (err.code) {
    case 'invalid_credentials':
      return 'invalid';
    case 'totp_invalid':
      return 'totpInvalid';
    case 'locked':
      return 'locked';
    case offline:
      return 'offline';
    case 'csrf':
    case 'bad_request':
      return 'stale';
    default:
      return 'server';
  }
}

export function useSignIn() {
  const [step, setStep] = useState<Step>('credentials');
  const [fields, setFields] = useState<Fields>({ username: '', password: '', code: '' });
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<SignInError | null>(null);
  const [lockedUntil, setLockedUntil] = useState<number | null>(null);

  // Fetches the token that guards this form; a live session goes straight in.
  useEffect(() => {
    getSession()
      .then((s) => {
        if (s.authenticated) enterApp();
      })
      .catch((e: unknown) => setError(classify(e)));
  }, []);

  const set = (name: keyof Fields, value: string) => setFields((f) => ({ ...f, [name]: value }));

  async function attempt(retried = false): Promise<void> {
    try {
      await signIn({
        username: fields.username.trim(),
        password: fields.password,
        totpCode: step === 'totp' ? fields.code : '',
      });
      enterApp();
    } catch (e) {
      if (e instanceof ApiError && e.code === 'totp_required') {
        setStep('totp');
        return;
      }
      const kind = classify(e);
      // An expired form token is refreshed once without bothering the operator.
      if (kind === 'stale' && !retried) {
        await getSession().catch(() => undefined);
        return attempt(true);
      }
      if (kind === 'locked' && e instanceof ApiError) {
        setLockedUntil(Date.now() + (e.retryAfter ?? 60) * 1000);
      }
      if (kind === 'totpInvalid') set('code', '');
      setError(kind);
    }
  }

  async function submit() {
    if (pending) return;
    if (!fields.username.trim() || !fields.password) {
      setError('required');
      return;
    }
    setPending(true);
    setError(null);
    await attempt();
    setPending(false);
  }

  function back() {
    setStep('credentials');
    setError(null);
    setFields((f) => ({ ...f, password: '', code: '' }));
  }

  return {
    step,
    fields,
    set,
    pending,
    error,
    lockedUntil,
    submit,
    back,
    clearLock: () => {
      setLockedUntil(null);
      setError(null);
    },
  };
}
