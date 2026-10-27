import { Select } from '@cloudflare/kumo/components/select';
import { useId } from 'react';
import { useT } from '../../../i18n/locale';
import type { AuditFilter } from '../../../lib/api/account';
import { auditMessages } from './messages';
import { actionMessages, auditActions } from './words';

const all = 'all';

/** The action and outcome filters above the log. */
export function AuditFilters({
  filter,
  onChange,
}: {
  filter: AuditFilter;
  onChange: (f: AuditFilter) => void;
}) {
  const t = useT(auditMessages);
  const ta = useT(actionMessages);
  const id = useId();
  const actions = Object.fromEntries([
    [all, t('allActions')],
    ...auditActions.map((a) => [a, ta(a)]),
  ]) as Record<string, string>;
  const outcomes = { [all]: t('allOutcomes'), success: t('success'), failure: t('failure') };
  return (
    <div className="flex flex-col gap-2 sm:flex-row">
      <span id={`${id}-action`} className="sr-only">
        {t('action')}
      </span>
      <Select
        aria-labelledby={`${id}-action`}
        className="w-full min-w-0 sm:w-72 pointer-coarse:h-9"
        value={filter.action || all}
        items={actions}
        onValueChange={(v) => onChange({ ...filter, action: !v || v === all ? '' : v })}
      />
      <span id={`${id}-outcome`} className="sr-only">
        {t('outcome')}
      </span>
      <Select
        aria-labelledby={`${id}-outcome`}
        className="w-full min-w-0 sm:w-44 pointer-coarse:h-9"
        value={filter.outcome || all}
        items={outcomes}
        onValueChange={(v) =>
          onChange({ ...filter, outcome: v === 'success' || v === 'failure' ? v : '' })
        }
      />
    </div>
  );
}
