import { Select } from '@cloudflare/kumo/components/select';
import { useId, useState } from 'react';
import { useT } from '../../i18n/locale';
import { analyticsMessages } from './messages';

export type Grouping = 'groupCore' | 'groupDirection';

/** Whether a chart shows its parts (grouped) or the one line they add up to. */
export function useGrouping() {
  return useState(true);
}

/** The "Group by" select at the end of a chart card's title row. */
export function GroupBy({
  by,
  grouped,
  onChange,
}: {
  by: Grouping;
  grouped: boolean;
  onChange: (grouped: boolean) => void;
}) {
  const t = useT(analyticsMessages);
  const id = useId();
  return (
    <span className="flex items-center gap-2">
      <span id={id} className="hidden text-kumo-subtle sm:inline">
        {t('groupBy')}
      </span>
      <Select
        aria-labelledby={id}
        size="sm"
        className="h-7 min-w-24 gap-1.5 px-2 text-sm"
        value={grouped ? by : 'none'}
        items={{ [by]: t(by), none: t('groupNone') }}
        onValueChange={(v) => onChange(v === by)}
      />
    </span>
  );
}
