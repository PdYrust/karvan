import type { NewJob } from '../../../lib/api/migrations';
import type { Panel, PanelKind } from '../../../lib/api/panels';

export type SignIn = 'password' | 'apiKey';

/** What the Connect form holds. Lives in component state only, never in storage. */
export interface ConnectValues {
  panelId: number;
  url: string;
  kind: 'auto' | PanelKind;
  signIn: SignIn;
  username: string;
  password: string;
  apiKey: string;
}

export const emptyValues: ConnectValues = {
  panelId: 0,
  url: '',
  kind: 'auto',
  signIn: 'password',
  username: '',
  password: '',
  apiKey: '',
};

/** The address as Karvan should call it; one typed without a scheme gets https://. */
export function sourceUrl(typed: string): string {
  const url = typed.trim();
  if (!url || /^[a-z][a-z0-9+.-]*:\/\//i.test(url)) return url;
  return `https://${url}`;
}

/** The request that creates the job; `pin` is the one certificate the operator trusted. */
export function toNewJob(v: ConnectValues, pin?: string): NewJob {
  const job: NewJob = { panelId: v.panelId, sourceUrl: sourceUrl(v.url) };
  if (v.signIn === 'apiKey') {
    // An API key only signs in to PasarGuard.
    job.sourceKind = 'pasarguard';
    job.apiKey = v.apiKey.trim();
  } else {
    if (v.kind !== 'auto') job.sourceKind = v.kind;
    job.username = v.username.trim();
    job.password = v.password;
  }
  if (pin) job.pinSha256 = pin;
  return job;
}

export type MissingField = 'panel' | 'url' | 'username' | 'password' | 'apiKey';

/** The first field the form still needs, or null when it can be sent. */
export function missingField(v: ConnectValues): MissingField | null {
  if (v.panelId <= 0) return 'panel';
  if (!v.url.trim()) return 'url';
  if (v.signIn === 'apiKey') return v.apiKey.trim() ? null : 'apiKey';
  if (!v.username.trim()) return 'username';
  return v.password ? null : 'password';
}

/** The destination to start with: the current panel when it is ready, else the first ready one. */
export function defaultDestination(panels: readonly Panel[], current: Panel | null): number {
  if (current?.ready) return current.id;
  return panels.find((p) => p.id > 0 && p.ready)?.id ?? 0;
}

/** The facts of a certificate Karvan does not trust, as the server sends them. */
export interface CertFacts {
  sha256: string;
  subject: string;
  notAfter: string;
}

export function certOf(code: string, detail: Readonly<Record<string, unknown>>): CertFacts | null {
  if (code !== 'source_tls_untrusted') return null;
  const { sha256, subject, notAfter } = detail;
  if (typeof sha256 !== 'string' || !/^[0-9a-f]{64}$/i.test(sha256)) return null;
  return {
    sha256: sha256.toLowerCase(),
    subject: typeof subject === 'string' ? subject : '',
    notAfter: typeof notAfter === 'string' ? notAfter : '',
  };
}

/** "AB:12:..." as browsers show a fingerprint, so the two can be compared. */
export function fingerprintPairs(sha256: string): string {
  return (sha256.toUpperCase().match(/.{1,2}/g) ?? []).join(':');
}
