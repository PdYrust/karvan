import { Button } from '@cloudflare/kumo/components/button';
import { Input } from '@cloudflare/kumo/components/input';
import { Select } from '@cloudflare/kumo/components/select';
import { useState, type FormEvent } from 'react';
import { useT } from '../../../i18n/locale';
import type { ProblemGroup, UserRow } from '../../../lib/api/migrations';
import { setUserFix, userFix } from '../../../lib/plan/plan';
import { useWizard } from '../wizard-context';
import { conflictMessages } from './conflict-messages';
import { compareMessages } from './messages';
import { appliedFix, fixWords, userFixes } from './problems';

const follow = 'follow';

/** One user's fix for one conflict: the rest's fix, or one of their own (a new name with its field). */
export function UserFix({ group, row }: { group: ProblemGroup; row: UserRow }) {
  const t = useT(compareMessages);
  const tc = useT(conflictMessages);
  const { plan } = useWizard();
  const own = plan.plan ? userFix(plan.plan, row.key, group.type) : undefined;
  const [picked, setPicked] = useState<string | null>(null);
  const saved = own?.option === 'rename' ? (own.params?.username ?? '') : '';
  const [name, setName] = useState(saved || row.username);
  const value = picked ?? own?.option ?? follow;
  const applied = appliedFix(row, group.type);
  const items: Record<string, string> = {
    [follow]: applied
      ? `${t('followAll')}: ${tc(fixWords(applied), { fix: applied })}`
      : t('followAll'),
    ...Object.fromEntries(userFixes(group).map((o) => [o.id, tc(fixWords(o.id), { fix: o.id })])),
  };
  const choose = (v: string) => {
    if (v === 'rename') return setPicked(v);
    setPicked(null);
    plan.edit(setUserFix(row.key, group.type, v === follow ? null : { option: v }));
  };
  const rename = (e: FormEvent) => {
    e.preventDefault();
    setPicked(null);
    plan.edit(
      setUserFix(row.key, group.type, { option: 'rename', params: { username: name.trim() } }),
    );
  };
  const label = t('userFix', { name: row.sourceUsername });
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Select
        aria-label={label}
        className="w-full sm:w-64 pointer-coarse:h-11"
        value={value}
        items={items}
        onValueChange={(v) => v && choose(v)}
      />
      {value === 'rename' ? (
        <form onSubmit={rename} className="flex gap-2">
          <Input
            aria-label={t('newName')}
            dir="ltr"
            autoComplete="off"
            spellCheck={false}
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-40 pointer-coarse:h-11"
          />
          <Button
            type="submit"
            disabled={!name.trim() || name.trim() === saved}
            className="pointer-coarse:h-11"
          >
            {t('save')}
          </Button>
        </form>
      ) : null}
    </div>
  );
}
