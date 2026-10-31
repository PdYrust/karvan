import { api } from './client';

export type PanelKind = 'marzban' | 'pasarguard';
export type DatabaseFlavor = 'sqlite' | 'mysql' | 'mariadb' | 'postgres';

/** Values Karvan read from the panel's ENV. Never holds a password. */
export interface PanelSettings {
  xrayConfig?: string;
  excludeInbounds?: string[];
  subPath: string;
  subUrlPrefix?: string;
  host: string;
  port: number;
  uds?: string;
  certFile?: string;
  keyFile?: string;
}

/** Something that stops Karvan from using the panel, with its code. */
export interface Finding {
  code: string;
  detail?: string;
}

export interface PanelControl {
  available: boolean;
  reason?: string;
}

export interface Panel {
  /** 0 for a folder a scan saw but could not use. */
  id: number;
  name: string;
  dir: string;
  envPath: string;
  kind?: PanelKind;
  hint?: PanelKind;
  revision?: string;
  /** Address of the database without its password. */
  db?: string;
  flavor?: DatabaseFlavor;
  settings: PanelSettings;
  findings: Finding[];
  ready: boolean;
  control?: PanelControl;
}

interface PanelList {
  panels: Panel[];
}

/** Every registered panel, inspected now. `fresh` also probes panel control again instead of using the last minute's answer. */
export async function getPanels(fresh = false): Promise<Panel[]> {
  return (await api<PanelList>('GET', fresh ? 'panels?fresh=1' : 'panels')).panels;
}

export async function scanPanels(): Promise<Panel[]> {
  return (await api<PanelList>('POST', 'panels/scan')).panels;
}

/** Releases Karvan supports, by database revision. */
const releases: Record<string, string> = {
  '2b231de97dc3': '0.8.4',
  '48a6bcb8bba1': '5.4.1',
  c8d2e4f6a1b3: '5.5.1',
};

/** "5.4.1" for a known revision, undefined otherwise. */
export function panelRelease(revision: string | undefined): string | undefined {
  return revision ? releases[revision] : undefined;
}

/** The kind to show: what the database says, else what the compose file suggests. */
export function panelKind(p: Pick<Panel, 'kind' | 'hint'>): PanelKind | undefined {
  return p.kind ?? p.hint;
}

// Engine names stay in Latin in both languages.
const engineNames: Readonly<Record<string, string>> = {
  sqlite: 'SQLite',
  mysql: 'MySQL',
  mariadb: 'MariaDB',
  postgres: 'PostgreSQL',
};

/** "PostgreSQL" for "postgres"; an engine this list does not know keeps its own name. */
export function engineName(flavor: string): string {
  return engineNames[flavor] ?? flavor;
}

const schemes: Readonly<Record<string, DatabaseFlavor>> = {
  sqlite: 'sqlite',
  mysql: 'mysql',
  postgres: 'postgres',
};

/**
 * Where a panel's database lives, read from its address: the file of a
 * SQLite database, the socket or host and port of a server. The engine is
 * the one the server reported, or the address scheme when it could not be
 * reached (MariaDB then reads as MySQL).
 */
export function databaseOf(p: Pick<Panel, 'db' | 'flavor'>): {
  engine: DatabaseFlavor | undefined;
  where: string | undefined;
} {
  const [, scheme, rest] = /^(\w+):\/\/(.+)$/.exec(p.db ?? '') ?? [];
  const engine = p.flavor ?? (scheme ? schemes[scheme] : undefined);
  if (!scheme || !rest) return { engine, where: undefined };
  if (scheme === 'sqlite') return { engine, where: rest };
  // user@unix(/path)/db or user@host:port/db; the user and the database are left out.
  const socket = /@unix\((.+)\)\/[^/]*$/.exec(rest)?.[1];
  const host = /^(?:.*@)?([^@/]+)\//.exec(rest)?.[1];
  return { engine, where: socket ?? host };
}
