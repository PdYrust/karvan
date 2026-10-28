import { describe, expect, it } from 'vitest';
import { filterCommands, type CommandGroup } from './commands';
import { batchKeywords, siteKeywords } from './job-commands';

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

describe('siteKeywords', () => {
  const site = {
    id: 3,
    name: 'Old Iran links',
    panelId: 2,
    sourceKind: 'marzban',
    sourceUrl: 'https://old.example.com:8443',
    oldPath: 'sub',
    domains: ['sub.example.net'],
    port: 2096,
    hasKey: true,
    lazy: false,
    plainHttpOk: false,
  } as const;
  const sites: CommandGroup[] = [
    {
      id: 'sites',
      label: 'Old links',
      searchOnly: true,
      items: [item('site:3', site.name, siteKeywords({ ...site, domains: [...site.domains] }))],
    },
  ];
  it('finds a site by its name, old address, domain, port and old panel', () => {
    for (const q of ['iran', 'old.example.com', 'sub.example.net', '2096', 'marzban', 'مرزبان']) {
      expect(filterCommands(sites, q).map((g) => g.id)).toEqual(['sites']);
    }
  });
  it('lists no site before something is typed', () => {
    expect(filterCommands(sites, '')).toEqual([]);
  });
});

describe('batchKeywords', () => {
  it('finds a batch by its name, admins, old address and panels', () => {
    const words = batchKeywords({
      id: 4,
      name: 'resellers',
      state: 'active',
      inactiveReason: '',
      inactiveAt: null,
      jobId: 'j1',
      panel: { id: 2, name: 'pg-main', kind: 'pasarguard' },
      source: { kind: 'marzban', origin: 'https://old.example.com' },
      admins: [{ source: 'ali', dest: 'ali_2', users: 3 }],
      oldLinks: { path: '', hosts: [], port: 0 },
      counts: { users: 3, verified: 0, closed: 0 },
      rules: [],
      createdAt: '2026-10-25T00:00:00Z',
      updatedAt: '2026-10-25T00:00:00Z',
    });
    for (const w of ['resellers', 'ali_2', 'old.example.com', 'pg-main', 'مرزبان']) {
      expect(words).toContain(w);
    }
  });
});
