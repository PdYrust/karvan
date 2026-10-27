import { Button } from '@cloudflare/kumo/components/button';
import { TrashIcon } from '@phosphor-icons/react';
import { useId, type ReactNode } from 'react';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { StoredTemplate } from '../../lib/plan/changes';
import type { Op } from '../../lib/plan/plan';
import { useFormat } from '../../lib/preferences/region';
import { useCategoryTitle } from '../wizard/changes/category-title';
import { changesMessages } from '../wizard/changes/messages';
import { opWords } from '../wizard/changes/words';
import { conflictMessages } from '../wizard/compare/conflict-messages';
import { fixWords, typeWords } from '../wizard/compare/problems';
import { templatesMessages } from './messages';
import { templateLines, type TemplateLine } from './template-lines';

/** Each line in words: who it reaches, then what it does. */
function useLineWords(now: Date) {
  const t = useT(templatesMessages);
  const tc = useT(changesMessages);
  const tf = useT(conflictMessages);
  const fmt = useFormat();
  const title = useCategoryTitle();
  // Group ids stand for themselves: a template names no panel, so it has no group names.
  const ctx = { now: now.toISOString(), targetName: (id: string) => id };
  const ops = (list: Op[]) => list.map((op) => opWords(op, tc, fmt, ctx)).join(', ');
  const problem = (type: string) => tf(typeWords(type), { type });
  return (l: TemplateLine): { who: ReactNode; what?: string } => {
    switch (l.kind) {
      case 'category':
        return { who: title(l.node).shown, what: ops(l.ops) };
      case 'problem':
        return { who: t('problem', { problem: problem(l.type) }), what: ops(l.ops) };
      case 'fix':
        return {
          who: t('fix', {
            problem: problem(l.type),
            fix: tf(fixWords(l.option), { fix: l.option }),
          }),
        };
      case 'reset':
        return { who: t(`reset_${l.schedule}`) };
    }
  };
}

/** One template: its name, when it was saved, what it does in words, and Delete. */
export function TemplateCard({
  template,
  now,
  onDelete,
}: {
  template: StoredTemplate;
  now: Date;
  onDelete: () => void;
}) {
  const t = useT(templatesMessages);
  const fmt = useFormat();
  const id = useId();
  const words = useLineWords(now);
  const lines = templateLines(template.template);
  const name = template.template.name;
  const saved = new Date(template.updatedAt);
  return (
    <article
      aria-labelledby={id}
      className="flex flex-col rounded-lg bg-kumo-base ring ring-kumo-hairline"
    >
      <header className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-kumo-hairline py-2 ps-4 pe-2">
        <h2 id={id} className="min-w-0 font-semibold text-kumo-strong">
          <bdi>{name}</bdi>
        </h2>
        <time
          dateTime={saved.toISOString()}
          title={fmt.dateTime(saved)}
          className="text-sm text-kumo-subtle"
        >
          {t('saved', { time: fmt.relative(saved, now) })}
        </time>
        <Button
          variant="ghost"
          size="sm"
          icon={<TrashIcon size={16} />}
          aria-label={t('deleteNamed', { name })}
          className="ms-auto pointer-coarse:h-9"
          onClick={onDelete}
        >
          {t('delete')}
        </Button>
      </header>
      {lines.length === 0 ? (
        <p className="px-4 py-3 text-kumo-subtle">{t('doesNothing')}</p>
      ) : (
        <ul className="flex flex-col">
          {lines.map((l, i) => {
            const w = words(l);
            return (
              <li
                // Lines have no id; their place in the template is who they are.
                key={`${l.kind}-${String(i)}`}
                className="flex flex-col gap-0.5 border-t border-kumo-hairline px-4 py-2.5 first:border-0 sm:flex-row sm:gap-4"
              >
                <span className="font-medium text-kumo-strong sm:w-2/5">{w.who}</span>
                {w.what ? (
                  <span className="text-kumo-default">
                    <CodeText text={w.what} />
                  </span>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </article>
  );
}
