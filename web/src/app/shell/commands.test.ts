import { describe, expect, it } from 'vitest';
import { filterCommands, type CommandGroup } from './commands';

const item = (id: string, title: string, keywords?: string) => ({
  id,
  title,
  keywords,
  icon: null,
  run: () => undefined,
});

const groups: CommandGroup[] = [
  { id: 'pages', label: 'Pages', items: [item('page:/', 'Overview')] },
  {
    id: 'migrations',
    label: 'Migrations',
    searchOnly: true,
    items: [item('job:1', 'reseller1 on old.example.com', 'reseller1 https://old.example.com')],
  },
];

describe('filterCommands', () => {
  it('leaves the latest migrations out until something is typed', () => {
    expect(filterCommands(groups, ' ').map((g) => g.id)).toEqual(['pages']);
  });
  it('finds a migration by its admin or address', () => {
    expect(filterCommands(groups, 'RESELLER').map((g) => g.id)).toEqual(['migrations']);
    expect(filterCommands(groups, 'old.example').map((g) => g.id)).toEqual(['migrations']);
    expect(filterCommands(groups, 'nothing like it')).toEqual([]);
  });
});
