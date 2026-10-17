/** An API failure with the server's error code. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryAfter: number | undefined;
  /** Every other field the server put next to the code, such as `confirm` or `reason`. */
  readonly detail: Readonly<Record<string, unknown>>;

  constructor(
    status: number,
    code: string,
    retryAfter?: number,
    detail: Record<string, unknown> = {},
  ) {
    super(code);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.retryAfter = retryAfter;
    this.detail = detail;
  }
}

/** Code used when the server cannot be reached at all. */
export const offline = 'offline';

let csrfToken = '';

/** Remembers the token every state-changing request must carry. */
export function setCsrfToken(token: string): void {
  csrfToken = token;
}

// document.baseURI includes the hidden path the server wrote into <base>.
export function apiURL(path: string): string {
  return new URL(`api/v1/${path}`, document.baseURI).toString();
}

interface ErrorBody {
  error?: Record<string, unknown>;
}

async function toError(res: Response): Promise<ApiError> {
  let body: ErrorBody = {};
  try {
    body = (await res.json()) as ErrorBody;
  } catch {
    // A non-JSON error (a proxy page, say) still gets a code below.
  }
  const { code, retryAfter, ...detail } = body.error ?? {};
  return new ApiError(
    res.status,
    typeof code === 'string' ? code : 'internal',
    typeof retryAfter === 'number' ? retryAfter : undefined,
    detail,
  );
}

export type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

export interface RequestOptions {
  /** Sent as If-Match, for writes that must not overwrite a newer version. */
  ifMatch?: string | number;
  /**
   * Polling that the operator did not ask for. The server then leaves the
   * session's idle timer alone, so an open dashboard still signs out.
   */
  background?: boolean;
  /** A body that is not JSON, such as an uploaded text file. */
  raw?: Blob | string;
  signal?: AbortSignal;
}

/** Sends one request to Karvan's API; a failure becomes an ApiError. */
export async function request(
  method: Method,
  path: string,
  body?: unknown,
  opts: RequestOptions = {},
): Promise<Response> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  let payload: BodyInit | null = null;
  if (opts.raw !== undefined) {
    payload = opts.raw;
    headers['Content-Type'] = 'text/plain; charset=utf-8';
  } else if (body !== undefined) {
    payload = JSON.stringify(body);
    headers['Content-Type'] = 'application/json';
  }
  if (method !== 'GET') headers['X-CSRF-Token'] = csrfToken;
  if (opts.ifMatch !== undefined) headers['If-Match'] = `"${String(opts.ifMatch)}"`;
  if (opts.background) headers['X-Karvan-Background'] = '1';

  let res: Response;
  try {
    res = await fetch(apiURL(path), {
      method,
      headers,
      credentials: 'same-origin',
      body: payload,
      signal: opts.signal ?? null,
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw new ApiError(0, offline);
  }
  if (!res.ok) throw await toError(res);
  return res;
}

export async function api<T>(
  method: Method,
  path: string,
  body?: unknown,
  opts?: RequestOptions,
): Promise<T> {
  const res = await request(method, path, body, opts);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
