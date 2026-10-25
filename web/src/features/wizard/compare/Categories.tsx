import { cn } from '@cloudflare/kumo/utils';
import { CaretRightIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { Section } from '../../../components/layout/Section';
import { useT } from '../../../i18n/locale';
import { childrenOf, pathTo, type PlanNode } from '../../../lib/plan/plan';
import { useFormat } from '../../../lib/preferences/region';
import { useWizard } from '../wizard-context';
import { AddCategory, CategoryMenu } from './CategoryMenus';
import { compareMessages } from './messages';
import { UserList } from '../UserList';

interface TreeProps {
  tree: readonly PlanNode[];
  counts: Readonly<Record<string, number>>;
  total: number;
  open: string | undefined;
  onOpen: (id: string | undefined) => void;
}

const row =
  'flex min-h-10 w-full items-center gap-2 rounded-md px-2 text-start pointer-coarse:min-h-11';

/** Desktop: every category nested under All users, with its count, next to the list. */
function Tree({ tree, counts, total, open, onOpen }: TreeProps) {
  const t = useT(compareMessages);
  const fmt = useFormat();
  // Each level is a list inside its parent's row, indented once more.
  const branch = (parent: string | undefined) =>
    childrenOf(tree, parent).map((n) => (
      <li key={n.id} className="flex flex-col">
        <div className="flex items-center gap-1">
          <button
            type="button"
            aria-current={open === n.id ? 'true' : undefined}
            onClick={() => onOpen(n.id)}
            className={cn(row, 'hover:bg-kumo-tint', open === n.id && 'bg-kumo-tint font-medium')}
          >
            <bdi className="min-w-0 flex-1 truncate">{n.name}</bdi>
            <span className="text-sm text-kumo-subtle tabular">
              {fmt.number(counts[n.id] ?? 0)}
            </span>
          </button>
          <CategoryMenu node={n} onRemoved={() => onOpen(n.parentId)} />
        </div>
        {childrenOf(tree, n.id).length ? <ul className="ps-4">{branch(n.id)}</ul> : null}
      </li>
    ));
  return (
    <nav aria-label={t('categoryList')} className="hidden w-72 shrink-0 lg:block">
      <ul className="flex flex-col gap-0.5">
        <li>
          <button
            type="button"
            aria-current={open === undefined ? 'true' : undefined}
            onClick={() => onOpen(undefined)}
            className={cn(
              row,
              'hover:bg-kumo-tint',
              open === undefined && 'bg-kumo-tint font-medium',
            )}
          >
            <span className="flex-1">{t('allUsers')}</span>
            <span className="text-sm text-kumo-subtle tabular">{fmt.number(total)}</span>
          </button>
        </li>
        <li>
          <ul className="ps-4">{branch(undefined)}</ul>
        </li>
      </ul>
    </nav>
  );
}

/** Phone: where you are as a breadcrumb, then the categories inside it to drill into. */
function Drill({ tree, counts, open, onOpen }: Omit<TreeProps, 'total'>) {
  const t = useT(compareMessages);
  const fmt = useFormat();
  const path = open ? pathTo(tree, open) : [];
  const here = path.at(-1);
  const inside = childrenOf(tree, open);
  return (
    <div className="flex flex-col gap-3 lg:hidden">
      <nav aria-label={t('breadcrumb')}>
        <ol className="flex flex-wrap items-center gap-1 text-sm">
          {[undefined, ...path].map((n, i) => (
            <li key={n?.id ?? ''} className="flex items-center gap-1">
              {i > 0 ? (
                <CaretRightIcon
                  size={12}
                  aria-hidden="true"
                  className="text-kumo-subtle rtl:-scale-x-100"
                />
              ) : null}
              {n === here && (n || !open) ? (
                <span aria-current="location" className="px-1 font-medium text-kumo-strong">
                  <bdi>{n?.name ?? t('allUsers')}</bdi>
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => onOpen(n?.id)}
                  className="min-h-11 rounded-md px-1 text-kumo-link underline"
                >
                  <bdi>{n?.name ?? t('allUsers')}</bdi>
                </button>
              )}
            </li>
          ))}
        </ol>
      </nav>
      {here ? (
        <div className="flex justify-end">
          <CategoryMenu node={here} onRemoved={() => onOpen(here.parentId)} />
        </div>
      ) : null}
      {inside.length ? (
        <ul className="flex flex-col rounded-lg ring ring-kumo-hairline">
          {inside.map((n) => (
            <li key={n.id} className="border-t border-kumo-hairline first:border-0">
              <button
                type="button"
                onClick={() => onOpen(n.id)}
                className={cn(row, 'min-h-12 px-3')}
              >
                <bdi className="min-w-0 flex-1 truncate">{n.name}</bdi>
                <span className="text-sm text-kumo-subtle tabular">
                  {fmt.number(counts[n.id] ?? 0)}
                </span>
                <CaretRightIcon
                  size={14}
                  aria-hidden="true"
                  className="text-kumo-subtle rtl:-scale-x-100"
                />
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

/** Healthy users grouped into categories by ready-made conditions, nested, with their users. */
export function Categories({
  counts,
  total,
}: {
  counts: Readonly<Record<string, number>>;
  total: number;
}) {
  const t = useT(compareMessages);
  const { jobId, plan } = useWizard();
  const tree = plan.plan?.tree ?? [];
  const [picked, setPicked] = useState<string | undefined>();
  // A category another tab removed falls back to All users.
  const open = picked && tree.some((n) => n.id === picked) ? picked : undefined;
  const node = tree.find((n) => n.id === open);
  return (
    <Section title={t('categoriesTitle')} actions={<AddCategory parent={node} />}>
      <div className="flex flex-col gap-4 px-4 pb-4">
        <p className="text-kumo-subtle">{t('categoriesIntro')}</p>
        {tree.length === 0 ? (
          <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('noCategories')}</p>
        ) : null}
        <div className="flex flex-col gap-4 lg:flex-row lg:gap-6">
          {tree.length ? (
            <Tree tree={tree} counts={counts} total={total} open={open} onOpen={setPicked} />
          ) : null}
          {tree.length ? (
            <Drill tree={tree} counts={counts} open={open} onOpen={setPicked} />
          ) : null}
          <div className="min-w-0 flex-1">
            <UserList
              key={open ?? ''}
              jobId={jobId}
              group={open ? `category:${open}` : ''}
              label={node?.name ?? t('allUsers')}
              empty={t('emptyCategory')}
            />
          </div>
        </div>
      </div>
    </Section>
  );
}
