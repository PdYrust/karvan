import { startOfDay, type Day } from '../../../lib/format/date';
import {
  filterCondition,
  signedAmount,
  type FilterKey,
  type Scope,
} from '../../../lib/plan/changes';
import type { Op } from '../../../lib/plan/plan';

/** The operations the Changes step offers, in the order its list shows them. */
export const opKinds = [
  'traffic.add',
  'traffic.set',
  'traffic.unlimited',
  'traffic.reset_used',
  'time.add',
  'time.set',
  'time.restart',
  'time.unlimited',
  'status.set',
  'name.prefix',
  'name.suffix',
  'name.replace',
  'note.set',
  'note.append',
  'note.clear',
  'place.targets',
  'skip',
  'cred.new',
] as const;

export type OpKind = (typeof opKinds)[number];

/** Everything the operation fields hold; each kind reads its own part. */
export interface OpForm {
  amount: string;
  date: Day | null;
  status: 'active' | 'disabled';
  text: string;
  from: string;
  to: string;
  ids: readonly string[];
  protocols: readonly string[];
  /** The operator read the new credentials warning. */
  understood: boolean;
}

export const emptyOpForm: OpForm = {
  amount: '',
  date: null,
  status: 'active',
  text: '',
  from: '',
  to: '',
  ids: [],
  protocols: [],
  understood: false,
};

/** A positive amount: a signed one that is above 0. */
function positive(text: string, whole: boolean): string | null {
  const v = signedAmount(text, whole);
  return v !== null && Number(v) > 0 ? v : null;
}

const op = (kind: string, params?: Record<string, string | null>): Op | null => {
  if (!params) return { kind };
  if (Object.values(params).some((v) => v === null)) return null;
  return { kind, params: params as Record<string, string> };
};

/** The operation the fields describe, or null while they do not describe one yet. */
export function buildOp(kind: OpKind, f: OpForm, timeZone: string): Op | null {
  const text = f.text.trim() ? f.text : null;
  switch (kind) {
    case 'traffic.add':
      return op(kind, { gb: signedAmount(f.amount, false) });
    case 'traffic.set':
      return op(kind, { gb: positive(f.amount, false) });
    case 'time.add':
      return op(kind, { days: signedAmount(f.amount, true) });
    case 'time.restart':
      return op(kind, { days: positive(f.amount, true) });
    case 'time.set':
      return op(kind, {
        date: f.date ? startOfDay(f.date, timeZone).toISOString().replace('.000Z', 'Z') : null,
      });
    case 'status.set':
      return op(kind, { status: f.status });
    case 'name.prefix':
    case 'name.suffix':
    case 'note.set':
    case 'note.append':
      return op(kind, { text });
    case 'name.replace':
      return op(kind, { from: f.from.trim() ? f.from : null, to: f.to });
    case 'place.targets':
      return op(kind, { ids: f.ids.length ? f.ids.join(',') : null });
    case 'cred.new':
      return f.understood ? op(kind, { protocols: f.protocols.join(',') || null }) : null;
    default:
      return op(kind);
  }
}

/** Who a change reaches, as the fields hold it. */
export interface ScopeForm {
  kind: Scope['kind'];
  category: string;
  filter: FilterKey;
  value: string;
  keys: readonly string[];
}

export const emptyScopeForm: ScopeForm = {
  kind: 'all',
  category: '',
  filter: 'name_contains',
  value: '',
  keys: [],
};

/** The scope the fields describe, or null while they do not describe one yet. */
export function buildScope(f: ScopeForm): Scope | null {
  switch (f.kind) {
    case 'all':
      return { kind: 'all' };
    case 'category':
      return f.category ? { kind: 'category', id: f.category } : null;
    case 'filter': {
      const condition = filterCondition(f.filter, f.value);
      return condition ? { kind: 'filter', condition } : null;
    }
    case 'users':
      return f.keys.length ? { kind: 'users', keys: f.keys } : null;
  }
}
