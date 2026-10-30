import { api } from './client';
import { list } from './lists';
import type { PanelKind } from './panels';

/** Why a batch went inactive: a backup restore under way, done, or failed after it began. */
export type InactiveReason = 'restoring' | 'rolled_back' | 'restore_failed';

/** A finished migration's users, admins and old links (phase 4.1 D1, api-migration.md). */
export interface Batch {
  id: number;
  name: string;
  state: 'active' | 'inactive';
  /** Empty while the batch is active. */
  inactiveReason: InactiveReason | '';
  inactiveAt: string | null;
  /** The migration it came from. */
  jobId: string;
  /** The new panel. */
  panel: { id: number; name: string; kind: PanelKind | '' };
  /** The old panel. */
  source: { kind: PanelKind; origin: string };
  admins: { source: string; dest: string; users: number }[];
  /** Empty until a rule or the old panel names them. */
  oldLinks: { path: string; hosts: string[]; port: number };
  counts: { users: number; verified: number; closed: number };
  /** The rules that serve it. */
  rules: { id: number; name: string }[];
  createdAt: string;
  updatedAt: string;
}

export interface BatchPage {
  batches: Batch[];
  next?: string;
}

/** How a user came into the batch: moved by its run, already on the new panel, or from an earlier run. */
export type Joined = 'moved' | 'adopted' | 'carried';

/** One user of a batch. No old token ever leaves the server (D57). */
export interface BatchUser {
  /** The ledger row, for the new link. */
  id: number;
  sourceUsername: string;
  username: string;
  admin: string;
  destId: number;
  joined: Joined;
  oldLinks: { verified: number; lastChecked: string | null };
  closed: boolean;
}

export interface BatchUserPage {
  users: BatchUser[];
  next?: string;
}

export interface BatchFilter {
  panel?: number;
  state?: Batch['state'];
  cursor?: string;
  limit?: number;
}

function withQuery(path: string, q: URLSearchParams): string {
  const query = q.toString();
  return query ? `${path}?${query}` : path;
}

/** A batch with its lists, whatever the server wrote for an empty one. */
function batchOf(b: Batch): Batch {
  return {
    ...b,
    admins: list(b.admins),
    oldLinks: { ...b.oldLinks, hosts: list(b.oldLinks.hosts) },
    rules: list(b.rules),
  };
}

/** Batches newest first, a page at a time, never deleted ones. */
export async function listBatches(f: BatchFilter = {}): Promise<BatchPage> {
  const q = new URLSearchParams();
  if (f.panel) q.set('panel', String(f.panel));
  if (f.state) q.set('state', f.state);
  if (f.cursor) q.set('cursor', f.cursor);
  if (f.limit) q.set('limit', String(f.limit));
  const page = await api<BatchPage>('GET', withQuery('batches', q));
  return { ...page, batches: list(page.batches).map(batchOf) };
}

const batchPath = (id: number, rest = '') => `batches/${String(id)}${rest}`;

export async function getBatch(id: number): Promise<Batch> {
  return batchOf(await api<Batch>('GET', batchPath(id)));
}

/** A batch's users, a page at a time; the server searches old and new usernames. */
export async function listBatchUsers(
  id: number,
  f: { q?: string; cursor?: string; limit?: number } = {},
): Promise<BatchUserPage> {
  const q = new URLSearchParams();
  const term = f.q?.trim();
  if (term) q.set('q', term);
  if (f.cursor) q.set('cursor', f.cursor);
  if (f.limit) q.set('limit', String(f.limit));
  const page = await api<BatchUserPage>('GET', withQuery(batchPath(id, '/users'), q));
  return { ...page, users: list(page.users) };
}

export async function renameBatch(id: number, name: string): Promise<Batch> {
  return batchOf(await api<Batch>('PATCH', batchPath(id), { name }));
}

/** Marks an inactive batch deleted once its name is typed (D20). */
export function deleteBatch(id: number, confirm: string): Promise<void> {
  return api<undefined>('DELETE', batchPath(id), { confirm });
}

/** Only after a failed restore. */
export async function reactivateBatch(id: number): Promise<Batch> {
  return batchOf(await api<Batch>('POST', batchPath(id, '/reactivate')));
}

/** A user's new subscription link, read from the new panel on each call and never kept. */
export async function showBatchLink(id: number, ledgerId: number): Promise<string> {
  const r = await api<{ url: string }>('POST', batchPath(id, `/users/${String(ledgerId)}/link`));
  return r.url;
}

export const batchKeys = {
  all: ['batches'] as const,
  list: (f: BatchFilter) => ['batches', 'list', f] as const,
  batch: (id: number) => ['batches', 'one', id] as const,
  users: (id: number, f: { q: string; cursor?: string }) =>
    ['batches', 'one', id, 'users', f] as const,
};
