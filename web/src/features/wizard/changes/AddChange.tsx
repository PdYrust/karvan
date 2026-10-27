import { Badge } from '@cloudflare/kumo/components/badge';
import { Button } from '@cloudflare/kumo/components/button';
import { Select } from '@cloudflare/kumo/components/select';
import { PlusIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { Section } from '../../../components/layout/Section';
import { useT } from '../../../i18n/locale';
import type { JobSummary } from '../../../lib/api/migrations';
import { addChange, changeTarget, filters, opFlags, type Scope } from '../../../lib/plan/changes';
import { useFormat } from '../../../lib/preferences/region';
import { useWizard } from '../wizard-context';
import {
  buildOp,
  buildScope,
  emptyOpForm,
  emptyScopeForm,
  opKinds,
  type OpKind,
  type ScopeForm,
} from './build';
import { changesMessages, type ChangesKey } from './messages';
import { OpFields } from './OpFields';
import { WhoFields } from './WhoFields';

const opWord = (k: OpKind) => `op_${k.replace('.', '_')}` as ChangesKey;

/** Who and what: the operator picks the users, then one operation and its values, then Add. */
export function AddChange({
  summary,
  counts,
}: {
  summary: JobSummary;
  counts: Readonly<Record<string, number>>;
}) {
  const t = useT(changesMessages);
  const fmt = useFormat();
  const { jobId, plan } = useWizard();
  const tree = plan.plan?.tree ?? [];
  const [who, setWho] = useState<ScopeForm>(emptyScopeForm);
  const [kind, setKind] = useState<OpKind>('traffic.add');
  const [form, setForm] = useState(emptyOpForm);
  // Counts the changes added, so the fields start over after each one.
  const [round, setRound] = useState(0);
  const scope = buildScope(who);
  const op = buildOp(kind, form, fmt.timeZone);
  const flags = opFlags(kind);
  // How many users the change reaches, when Karvan knows before it is added.
  const reach = (s: Scope | null): number | undefined => {
    if (s?.kind === 'all') return summary.users;
    if (s?.kind === 'category') return counts[s.id] ?? 0;
    if (s?.kind === 'users') return s.keys.length;
    return undefined;
  };
  const reached = reach(scope);
  const nodeName = (s: Scope) => {
    if (s.kind === 'filter') {
      const value = s.condition.list?.[0] ?? s.condition.value ?? '';
      const f = filters[who.filter].value === 'status' ? t(value as ChangesKey) : value;
      return t('filterName', { filter: t(`filter_${who.filter}` as ChangesKey), value: f });
    }
    return s.kind === 'users' ? t('pickedName', { count: s.keys.length }) : t('whoAll');
  };
  // Not a form: Enter in the user search must not add the change.
  const add = () => {
    if (!scope || !op) return;
    const { id, node } = changeTarget(scope, nodeName(scope), tree);
    plan.edit(addChange(id, node, op));
    setForm(emptyOpForm);
    setRound(round + 1);
    if (who.kind === 'users') setWho({ ...who, keys: [] });
  };
  return (
    <Section title={t('addTitle')}>
      <div className="flex flex-col gap-6 px-4 pb-4">
        <WhoFields jobId={jobId} tree={tree} counts={counts} form={who} onChange={setWho} />
        <div className="flex flex-col gap-4 border-t border-kumo-hairline pt-4">
          <div className="flex flex-col gap-2">
            <span id="change-kind" className="text-sm font-medium text-kumo-strong">
              {t('what')}
            </span>
            <div className="flex flex-wrap items-center gap-2">
              <Select
                aria-labelledby="change-kind"
                className="w-full sm:w-80 pointer-coarse:h-9"
                value={kind}
                items={Object.fromEntries(opKinds.map((k) => [k, t(opWord(k))]))}
                onValueChange={(k) => {
                  if (!k) return;
                  setKind(k as OpKind);
                  setForm(emptyOpForm);
                }}
              />
              {flags.config ? <Badge variant="outline">{t('configFlag')}</Badge> : null}
              {flags.link ? <Badge variant="outline">{t('linkFlag')}</Badge> : null}
            </div>
          </div>
          <OpFields
            key={`${kind}-${String(round)}`}
            kind={kind}
            summary={summary}
            form={form}
            onChange={setForm}
          />
        </div>
        <div className="flex flex-wrap items-center justify-end gap-3 border-t border-kumo-hairline pt-4">
          {reached !== undefined ? (
            <span className="text-sm text-kumo-subtle sm:me-auto" aria-live="polite">
              {t('reaches', { count: reached })}
            </span>
          ) : null}
          <Button
            onClick={add}
            icon={<PlusIcon size={16} />}
            disabled={!scope || !op}
            className="pointer-coarse:h-9"
          >
            {t('add')}
          </Button>
        </div>
      </div>
    </Section>
  );
}
