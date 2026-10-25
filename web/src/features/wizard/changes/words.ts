import type { Translate } from '../../../i18n/core';
import type { Format } from '../../../lib/format/format';
import type { Op } from '../../../lib/plan/plan';
import { protocolName } from '../protocols';
import type { ChangesKey } from './messages';

type T = Translate<ChangesKey>;

const statusWords = ['active', 'disabled', 'limited', 'expired', 'on_hold'] as const;

/** An operation in words for the list of changes, such as "Add 10 GB" or "Expires on 1 Farvardin 1406". */
export function opWords(
  op: Op,
  t: T,
  fmt: Format,
  ctx: { now: string; targetName: (id: string) => string },
): string {
  const p = op.params ?? {};
  const n = Number(p.gb ?? p.days ?? 0);
  const amount = fmt.number(Math.abs(n), 2);
  switch (op.kind) {
    case 'traffic.add':
      return t(n < 0 ? 'w_traffic_take' : 'w_traffic_add', { amount });
    case 'traffic.set':
      return n > 0 ? t('w_traffic_set', { amount }) : t('w_traffic_unlimited');
    case 'time.add':
      return t(n < 0 ? 'w_time_take' : 'w_time_add', { count: Math.abs(n) });
    case 'time.restart':
      return t('w_time_restart', { count: n, date: fmt.longDate(new Date(ctx.now)) });
    case 'time.set':
      return t('w_time_set', { date: fmt.longDate(new Date(p.date ?? '')) });
    case 'status.set': {
      const s = statusWords.find((w) => w === p.status);
      return t('w_status_set', { status: s ? t(s) : (p.status ?? '') });
    }
    case 'name.replace':
      return p.to
        ? t('w_name_replace', { from: p.from ?? '', to: p.to })
        : t('w_name_remove', { from: p.from ?? '' });
    case 'name.prefix':
    case 'name.suffix':
    case 'note.set':
    case 'note.append':
      return t(`w_${op.kind.replace('.', '_')}` as ChangesKey, { text: p.text ?? '' });
    case 'place.targets':
      return t('w_place_targets', {
        names: (p.ids ?? '').split(',').filter(Boolean).map(ctx.targetName).join(', '),
      });
    case 'cred.new':
      return t('w_cred_new', {
        protocols: (p.protocols ?? '').split(',').map(protocolName).join(', '),
      });
    case 'traffic.unlimited':
    case 'traffic.reset_used':
    case 'time.unlimited':
    case 'note.clear':
    case 'skip':
      return t(`w_${op.kind.replace('.', '_')}` as ChangesKey);
    default:
      return t('w_other', { kind: op.kind });
  }
}
