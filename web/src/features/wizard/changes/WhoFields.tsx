import { Button } from '@cloudflare/kumo/components/button';
import { Checkbox } from '@cloudflare/kumo/components/checkbox';
import { Input } from '@cloudflare/kumo/components/input';
import { Radio } from '@cloudflare/kumo/components/radio';
import { Select } from '@cloudflare/kumo/components/select';
import { useT } from '../../../i18n/locale';
import { filters, statuses, type FilterKey, type Scope } from '../../../lib/plan/changes';
import type { PlanNode } from '../../../lib/plan/plan';
import { useFormat } from '../../../lib/preferences/region';
import { UserList } from '../UserList';
import type { ScopeForm } from './build';
import { changesMessages, type ChangesKey } from './messages';

const filterWord = (k: FilterKey) => `filter_${k}` as ChangesKey;

function FilterFields({ form, onChange }: { form: ScopeForm; onChange: (f: ScopeForm) => void }) {
  const t = useT(changesMessages);
  const kind = filters[form.filter].value;
  return (
    <div className="flex flex-wrap items-end gap-3">
      <div className="flex flex-col gap-2">
        <span id="change-filter" className="text-sm font-medium text-kumo-strong">
          {t('filterLabel')}
        </span>
        <Select
          aria-labelledby="change-filter"
          className="w-64 pointer-coarse:h-11"
          value={form.filter}
          items={Object.fromEntries(
            (Object.keys(filters) as FilterKey[]).map((k) => [k, t(filterWord(k))]),
          )}
          onValueChange={(k) => k && onChange({ ...form, filter: k as FilterKey, value: '' })}
        />
      </div>
      <div className="flex flex-col gap-2">
        <span id="change-filter-value" className="text-sm font-medium text-kumo-strong">
          {t('valueLabel')}
        </span>
        {kind === 'status' ? (
          <Select
            aria-labelledby="change-filter-value"
            placeholder={t('choose')}
            className="w-44 pointer-coarse:h-11"
            value={form.value || null}
            items={Object.fromEntries(statuses.map((s) => [s, t(s)]))}
            onValueChange={(v) => onChange({ ...form, value: v ?? '' })}
          />
        ) : (
          <Input
            aria-labelledby="change-filter-value"
            dir="ltr"
            inputMode={kind === 'number' ? 'decimal' : 'text'}
            autoComplete="off"
            spellCheck={false}
            value={form.value}
            onChange={(e) => onChange({ ...form, value: e.target.value })}
            className="w-44 pointer-coarse:h-11"
          />
        )}
      </div>
    </div>
  );
}

/** Who a change reaches: everyone, a category, a filter, or users picked from the list. */
export function WhoFields({
  jobId,
  tree,
  counts,
  form,
  onChange,
}: {
  jobId: string;
  tree: readonly PlanNode[];
  counts: Readonly<Record<string, number>>;
  form: ScopeForm;
  onChange: (f: ScopeForm) => void;
}) {
  const t = useT(changesMessages);
  const fmt = useFormat();
  // Categories the changes made for a filter or picked users are not offered again here.
  const categories = tree.filter((n) => !n.id.startsWith('change-'));
  const kinds: [Scope['kind'], ChangesKey][] = [
    ['all', 'whoAll'],
    ['category', 'whoCategory'],
    ['filter', 'whoFilter'],
    ['users', 'whoUsers'],
  ];
  const picked = new Set(form.keys);
  const toggle = (key: string, on: boolean) =>
    onChange({
      ...form,
      keys: on ? [...form.keys, key] : form.keys.filter((k) => k !== key),
    });
  return (
    <div className="flex flex-col gap-4">
      <Radio.Group
        legend={t('who')}
        value={form.kind}
        onValueChange={(k) => onChange({ ...form, kind: k as Scope['kind'] })}
      >
        {kinds.map(([k, word]) => (
          <Radio.Item key={k} value={k} label={t(word)} />
        ))}
      </Radio.Group>
      {form.kind === 'category' ? (
        categories.length ? (
          <div className="flex flex-col gap-2">
            <span id="change-category" className="text-sm font-medium text-kumo-strong">
              {t('categoryLabel')}
            </span>
            <Select
              aria-labelledby="change-category"
              placeholder={t('choose')}
              className="w-full sm:w-72 pointer-coarse:h-11"
              value={form.category || null}
              items={Object.fromEntries(
                categories.map((n) => [n.id, `${n.name} (${fmt.number(counts[n.id] ?? 0)})`]),
              )}
              onValueChange={(id) => onChange({ ...form, category: id ?? '' })}
            />
          </div>
        ) : (
          <p className="text-kumo-subtle">{t('noCategoryYet')}</p>
        )
      ) : null}
      {form.kind === 'filter' ? <FilterFields form={form} onChange={onChange} /> : null}
      {form.kind === 'users' ? (
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-3" aria-live="polite">
            <span className="text-sm text-kumo-default">
              {t('picked', { count: form.keys.length })}
            </span>
            {form.keys.length ? (
              <Button
                variant="ghost"
                size="sm"
                className="pointer-coarse:h-11"
                onClick={() => onChange({ ...form, keys: [] })}
              >
                {t('clearPicked')}
              </Button>
            ) : null}
          </div>
          <UserList
            jobId={jobId}
            group=""
            label={t('whoUsers')}
            empty={t('noUsers')}
            end={(row) => (
              <Checkbox
                aria-label={t('pick', { name: row.sourceUsername })}
                checked={picked.has(row.key)}
                onCheckedChange={(on) => toggle(row.key, on)}
              />
            )}
          />
        </div>
      ) : null}
    </div>
  );
}
