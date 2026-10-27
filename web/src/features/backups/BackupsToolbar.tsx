import { useState } from 'react';
import { ListToolbar } from '../../components/data/ListToolbar';
import { dataMessages } from '../../components/data/messages';
import { sortChoice } from '../../components/data/sort-choice';
import { useT } from '../../i18n/locale';
import { matchesTerm, sortRows, type ListSort } from '../../lib/lists/choices';
import type { BackupRow } from './backup-rows';
import { backupsMessages } from './messages';
type Can = 'all' | 'ready' | 'locked';
const isCan = (r: BackupRow, c: Can) => c === 'all' || (c === 'ready') === (r.restore === null);
/** The backups list's search, filter and sort, and the rows they leave. */
export function useBackupFilters() {
  const [query, setQuery] = useState('');
  const [can, setCan] = useState<Can>('all');
  const [sort, setSort] = useState<ListSort>('newest');
  const shown = (rows: readonly BackupRow[]) =>
    sortRows(
      rows.filter(
        (r) =>
          isCan(r, can) &&
          matchesTerm(query, [
            r.backup.id,
            r.backup.sourceAdmin,
            r.backup.sourceOrigin,
            r.panel?.name,
          ]),
      ),
      sort,
      { time: (r) => r.backup.createdAt, size: (r) => r.backup.size },
      'en',
    );
  return { query, setQuery, can, setCan, sort, setSort, shown };
}
/** Search by admin, address, panel or backup id; can it be put back; newest or largest first. */
export function BackupsToolbar({
  rows,
  filters: f,
}: {
  rows: readonly BackupRow[];
  filters: ReturnType<typeof useBackupFilters>;
}) {
  const t = useT(backupsMessages);
  const td = useT(dataMessages);
  const text = { all: td('showAll'), ready: t('canPutBack'), locked: t('locked') };
  return (
    <ListToolbar
      search={{ label: t('search'), value: f.query, onChange: f.setQuery }}
      choices={[
        {
          label: td('filter'),
          value: f.can,
          all: 'all',
          options: (['all', 'ready', 'locked'] as const).map((c) => ({
            value: c,
            label: text[c],
            text: text[c],
            count: rows.filter((r) => isCan(r, c)).length,
          })),
          onChange: (c) => f.setCan(c as Can),
        },
        sortChoice(td, f.sort, f.setSort, ['newest', 'largest']),
      ]}
    />
  );
}
