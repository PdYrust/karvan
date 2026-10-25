import { Badge } from '@cloudflare/kumo/components/badge';
import { ArrowRightIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { useT } from '../../../i18n/locale';
import { usersPerPage } from '../../../lib/api/migrations';
import type { ChangeField, Changes, Outcome, UserChange } from '../../../lib/api/preflight';
import { startMessages } from './messages';
import { Pager } from '../../../components/data/Pager';
/** One field of an outcome in words, or as the machine values it holds. */
function useValue() {
  const t = useT(startMessages);
  return (field: ChangeField, o: Outcome): ReactNode => {
    const list = (xs: readonly string[] | null) =>
      xs && xs.length > 0 ? <bdi dir="ltr">{xs.join(', ')}</bdi> : t('none');
    switch (field) {
      case 'include':
        return o.include ? t('moves') : t('staysBehind');
      case 'adopted':
        return o.adopted ? t('field_adopted') : t('none');
      case 'username':
        return <bdi dir="ltr">{o.username}</bdi>;
      case 'targets':
        return list(o.targets);
      case 'credentials':
        return list(o.protocols);
      case 'conflicts':
        return list((o.conflicts ?? []).map((c) => c.type));
    }
  };
}
function Fields({ u }: { u: UserChange }) {
  const t = useT(startMessages);
  const value = useValue();
  const { before, after } = u;
  if (!before || !after) return null;
  return (
    <dl className="flex flex-col gap-1 text-sm">
      {(u.fields ?? []).map((f) => (
        <div key={f} className="flex flex-wrap items-center gap-x-2">
          <dt className="text-kumo-subtle">{t(`field_${f}`)}</dt>
          <dd className="flex flex-wrap items-center gap-x-1.5">
            {f === 'credentials' ? (
              t('newCredentials', { protocols: (after.protocols ?? []).join(', ') })
            ) : (
              <>
                {value(f, before)}
                <ArrowRightIcon size={12} aria-hidden="true" className="rtl:-scale-x-100" />
                {value(f, after)}
              </>
            )}
          </dd>
        </div>
      ))}
    </dl>
  );
}
const tones = { new: 'primary', gone: 'secondary', changed: 'outline' } as const;
/** Users new, gone or changed since the last read, 50 a page, each with before and after. */
export function ChangesDiff({
  changes,
  offset,
  onOffset,
}: {
  changes: Changes;
  offset: number;
  onOffset: (n: number) => void;
}) {
  const t = useT(startMessages);
  const { counts } = changes;
  const parts = [
    counts.new ? t('countNew', { count: counts.new }) : '',
    counts.gone ? t('countGone', { count: counts.gone }) : '',
    counts.changed ? t('countChanged', { count: counts.changed }) : '',
  ].filter(Boolean);
  if (changes.total === 0) return null;
  return (
    <div className="flex flex-col gap-3">
      <p className="flex flex-wrap gap-x-4 gap-y-1 font-medium text-kumo-strong">
        {parts.map((p) => (
          <span key={p}>{p}</span>
        ))}
      </p>
      <ul aria-label={t('diffLabel')} className="flex flex-col rounded-lg ring ring-kumo-hairline">
        {changes.users.map((u) => {
          const o = u.after ?? u.before;
          return (
            <li
              key={u.key}
              className="flex flex-col gap-1.5 border-t border-kumo-hairline px-3 py-2.5 first:border-0"
            >
              <span className="flex flex-wrap items-center gap-2">
                <bdi dir="ltr" className="font-medium text-kumo-strong">
                  {o?.sourceUsername ?? u.key}
                </bdi>
                <Badge variant={tones[u.change]}>{t(`change_${u.change}`)}</Badge>
              </span>
              <Fields u={u} />
            </li>
          );
        })}
      </ul>
      <Pager
        offset={offset}
        shown={changes.users.length}
        total={changes.total}
        perPage={usersPerPage}
        onOffset={onOffset}
      />
    </div>
  );
}
