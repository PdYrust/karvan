import { Badge } from '@cloudflare/kumo/components/badge';
import { Button } from '@cloudflare/kumo/components/button';
import { TrashIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { Section } from '../../../components/layout/Section';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { JobSummary } from '../../../lib/api/migrations';
import { filterOf, isChangeNode, opFlags, removeChange } from '../../../lib/plan/changes';
import type { PlanNode } from '../../../lib/plan/plan';
import { useFormat } from '../../../lib/preferences/region';
import { useWizard } from '../wizard-context';
import { changesMessages, type ChangesKey } from './messages';
import { opWords } from './words';

/**
 * A category's title as text (for its label) and as shown. The ones the
 * changes made read in the page's language; a filter's value is a machine
 * value kept left to right inside a Persian title.
 */
function useTitle() {
  const t = useT(changesMessages);
  return (n: PlanNode): { label: string; shown: ReactNode } => {
    const plain = (label: string) => ({ label, shown: <bdi>{label}</bdi> });
    if (!isChangeNode(n)) return plain(n.name);
    if (n.id === 'change-all') return plain(t('whoAll'));
    const f = filterOf(n);
    if (f) {
      const filter = t(`filter_${f.key}` as ChangesKey);
      const word = f.key === 'status';
      const value = word ? t(f.value as ChangesKey) : f.value;
      return {
        label: t('filterName', { filter, value }),
        shown: (
          <>
            {filter}: {word ? value : <bdi dir="ltr">{value}</bdi>}
          </>
        ),
      };
    }
    const users = n.conditions?.find((c) => c.field === 'users')?.list;
    return plain(users ? t('pickedName', { count: users.length }) : n.name);
  };
}

/** The changes already in the plan, by the category they reach, each with Remove. */
export function ChangeList({
  summary,
  counts,
}: {
  summary: JobSummary;
  counts: Readonly<Record<string, number>>;
}) {
  const t = useT(changesMessages);
  const fmt = useFormat();
  const { plan } = useWizard();
  const title = useTitle();
  const p = plan.plan;
  const nodes = (p?.tree ?? []).filter((n) => n.ops?.length);
  const names = new Map((summary.targets ?? []).map((g) => [g.id, g.name]));
  const ctx = { now: p?.now ?? '', targetName: (id: string) => names.get(id) ?? id };
  return (
    <Section title={t('listTitle')}>
      <div className="flex flex-col gap-4 px-4 pb-4">
        <p className="text-kumo-subtle">{t('listIntro')}</p>
        {nodes.length === 0 ? (
          <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('none')}</p>
        ) : (
          nodes.map((n) => (
            <section
              key={n.id}
              aria-label={title(n).label}
              className="flex flex-col rounded-lg ring ring-kumo-hairline"
            >
              <header className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 rounded-t-lg bg-kumo-tint px-3 py-2">
                <h3 className="font-medium text-kumo-strong">{title(n).shown}</h3>
                <span className="text-sm text-kumo-subtle">
                  {t('reaches', { count: counts[n.id] ?? 0 })}
                </span>
              </header>
              <ul>
                {(n.ops ?? []).map((op, i) => {
                  const words = opWords(op, t, fmt, ctx);
                  const flags = opFlags(op.kind);
                  return (
                    <li
                      // Operations have no id; their place in the list is who they are.
                      key={`${op.kind}-${String(i)}`}
                      className="flex min-h-12 flex-wrap items-center gap-x-3 gap-y-1 border-t border-kumo-hairline px-3 py-2"
                    >
                      <span className="min-w-0 text-kumo-default">
                        <CodeText text={words} />
                      </span>
                      {flags.config ? <Badge variant="outline">{t('configFlag')}</Badge> : null}
                      {flags.link ? <Badge variant="outline">{t('linkFlag')}</Badge> : null}
                      <Button
                        variant="ghost"
                        size="sm"
                        icon={<TrashIcon size={16} />}
                        aria-label={t('removeChange', { change: words.replaceAll('`', '') })}
                        className="ms-auto pointer-coarse:h-11"
                        onClick={() => plan.edit(removeChange(n.id, i))}
                      >
                        {t('remove')}
                      </Button>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))
        )}
      </div>
    </Section>
  );
}
