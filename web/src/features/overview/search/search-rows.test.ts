import { describe, expect, it } from 'vitest';
import type { Command, CommandGroup } from '../../../app/shell/commands';
import { moveActive, searchGroups, searchRecentsMax, searchRows } from './search-rows';
const command = (id: string, title = id): Command => ({ id, title, icon: null, run: () => id });
const recents: CommandGroup = {
  id: 'recents',
  label: 'Recents',
  items: ['a', 'b', 'c', 'd', 'e', 'f', 'g'].map((id) => command(id)),
};
const groups: CommandGroup[] = [
  {
    id: 'pages',
    label: 'Pages',
    items: [command('panels', 'Panels'), command('backups', 'Backups')],
  },
  { id: 'jobs', label: 'Migrations', items: [command('job', 'Marzban admin')], searchOnly: true },
];
describe('overview search rows', () => {
  it('lists only the latest recent pages while nothing is typed', () => {
    const shown = searchGroups(recents, groups, '  ');
    expect(shown.map((g) => g.id)).toEqual(['recents']);
    expect(shown[0]?.items).toHaveLength(searchRecentsMax);
  });
  it('lists the pages before typing when no page was opened yet', () => {
    const shown = searchGroups({ ...recents, items: [] }, groups, '');
    expect(shown.map((g) => g.id)).toEqual(['pages']);
  });
  it('finds what the palette finds once something is typed', () => {
    const shown = searchGroups(recents, groups, 'pan');
    expect(searchRows(shown).map((c) => c.id)).toEqual(['panels']);
    expect(searchRows(searchGroups(recents, groups, 'admin')).map((c) => c.id)).toEqual(['job']);
  });
  it('walks the rows with the arrow keys and wraps', () => {
    expect(moveActive(0, 'ArrowDown', 3)).toBe(1);
    expect(moveActive(2, 'ArrowDown', 3)).toBe(0);
    expect(moveActive(0, 'ArrowUp', 3)).toBe(2);
    expect(moveActive(1, 'End', 3)).toBe(2);
    expect(moveActive(1, 'Home', 3)).toBe(0);
    expect(moveActive(0, 'ArrowDown', 0)).toBeUndefined();
    expect(moveActive(0, 'Enter', 3)).toBeUndefined();
  });
});
