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

export async function getPanels(): Promise<Panel[]> {
  return (await api<PanelList>('GET', 'panels')).panels;
}

export async function scanPanels(): Promise<Panel[]> {
  return (await api<PanelList>('POST', 'panels/scan')).panels;
}

/** Releases Karvan supports, by database revision. */
const releases: Record<string, string> = {
  '2b231de97dc3': '0.8.4',
  '48a6bcb8bba1': '5.4.1',
};

/** "5.4.1" for a known revision, undefined otherwise. */
export function panelRelease(revision: string | undefined): string | undefined {
  return revision ? releases[revision] : undefined;
}

/** The kind to show: what the database says, else what the compose file suggests. */
export function panelKind(p: Pick<Panel, 'kind' | 'hint'>): PanelKind | undefined {
  return p.kind ?? p.hint;
}
