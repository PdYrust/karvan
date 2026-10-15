/** An API failure with the server's error code. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryAfter: number | undefined;

  constructor(status: number, code: string, retryAfter?: number) {
    super(code);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.retryAfter = retryAfter;
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
function url(path: string): string {
  return new URL(`api/v1/${path}`, document.baseURI).toString();
}

interface ErrorBody {
  error?: { code?: unknown; retryAfter?: unknown };
}

async function toError(res: Response): Promise<ApiError> {
  let body: ErrorBody = {};
  try {
    body = (await res.json()) as ErrorBody;
  } catch {
    // A non-JSON error (a proxy page, say) still gets a code below.
  }
  const code = typeof body.error?.code === 'string' ? body.error.code : 'internal';
  const retryAfter = typeof body.error?.retryAfter === 'number' ? body.error.retryAfter : undefined;
  return new ApiError(res.status, code, retryAfter);
}

export async function api<T>(
  method: 'GET' | 'POST' | 'DELETE',
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (method !== 'GET') headers['X-CSRF-Token'] = csrfToken;

  let res: Response;
  try {
    res = await fetch(url(path), {
      method,
      headers,
      credentials: 'same-origin',
      body: body === undefined ? null : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, offline);
  }
  if (!res.ok) throw await toError(res);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
