import { api } from './client';
import type { PanelKind } from './panels';

export const rangeKeys = ['live', '1h', '24h', '7d', '30d'] as const;
export type RangeKey = (typeof rangeKeys)[number];

export interface Memory {
  total: number;
  used: number;
  cache: number;
  available: number;
  swapTotal: number;
  swapUsed: number;
}

/** One physical network card; rates are bytes per second. */
export interface IfaceRate {
  name: string;
  rx: number;
  tx: number;
  totalRx: number;
  totalTx: number;
}

/** The server over the last 5 seconds. Shares are 0 to 1, rates bytes per second. */
export interface Sample {
  seq: number;
  at: string;
  cpu: number;
  cores: number[];
  load: [number, number, number];
  mem: Memory;
  netRx: number;
  netTx: number;
  ifaces: IfaceRate[];
  diskRead: number;
  diskWrite: number;
  tcp: number;
  tcpTimeWait: number;
  procs: number;
  procsRunning: number;
  uptime: number;
}

/** The server's average over a bucket, with its peaks. `t` is unix seconds. */
export interface HostMinute {
  t: number;
  cpu: number;
  cpuMax: number;
  load1: number;
  memUsed: number;
  memCache: number;
  memTotal: number;
  swapUsed: number;
  netRx: number;
  netTx: number;
  netRxMax: number;
  netTxMax: number;
  diskRead: number;
  diskWrite: number;
  tcp: number;
  procs: number;
  /** Saved minutes in the bucket; fewer than its width when the server was down for part of it. */
  minutes: number;
}

export interface HostFacts {
  hostname: string;
  os: string;
  kernel: string;
  arch: string;
  cpuModel: string;
  cores: number;
  bootedAt: string;
  karvanRss: number;
  startedAt: string;
}

export interface Disk {
  mount: string;
  type: string;
  total: number;
  used: number;
  free: number;
}

export interface ServerStats {
  /** False until the first sample, or where /proc cannot be read. */
  available: boolean;
  range: RangeKey;
  /** Seconds between samples (live) or buckets. */
  step: number;
  seq: number;
  facts: HostFacts;
  disks: Disk[];
  now: Sample | null;
  samples?: Sample[];
  points?: HostMinute[];
}

export interface EngineStats {
  flavor: string;
  version: string;
  size: number;
  connections: number;
  maxConnections: number;
  running: number;
  uptime: number;
  pageSize?: number;
  pages?: number;
  freePages?: number;
  walSize?: number;
  journalMode?: string;
  timescale?: string;
}

export interface AdminStats {
  username: string;
  users: number;
  active: number;
  usedTraffic: number;
}

export interface NodeStats {
  name: string;
  status: string;
  uplink: number;
  downlink: number;
}

export interface PanelData {
  users: number;
  byStatus: Record<string, number>;
  onlineNow: number;
  onlineDay: number;
  newDay: number;
  newWeek: number;
  expiringWeek: number;
  usedTraffic: number;
  admins: number;
  topAdmins: AdminStats[];
  nodes: NodeStats[];
  uplink: number;
  downlink: number;
}

/** A panel database over a bucket. `t` is unix seconds; cacheHit is null for SQLite. */
export interface DbMinute {
  t: number;
  size: number;
  connections: number;
  qps: number;
  cacheHit: number | null;
  users: number;
  active: number;
  online: number;
  usedTraffic: number;
}

export interface PanelStats {
  panelId: number;
  name: string;
  kind?: PanelKind;
  at: string;
  /** A finding that stops Karvan from reading the database. */
  problem?: { code: string };
  engine?: EngineStats;
  qps: number;
  cacheHit: number | null;
  data?: PanelData;
  /** Registered, but not read yet. */
  pending?: boolean;
  points: DbMinute[];
}

export interface TableSize {
  name: string;
  rows: number;
  bytes: number;
}

export interface TrafficHour {
  at: string;
  uplink: number;
  downlink: number;
}

export interface HourCount {
  at: string;
  n: number;
}

export interface ClientCount {
  name: string;
  count: number;
}

export interface PanelDetail {
  tables: TableSize[];
  traffic: TrafficHour[];
  signUps: HourCount[];
  clients: ClientCount[];
}

export interface PanelAnalytics {
  range: RangeKey;
  step: number;
  panel: PanelStats;
  detail?: PanelDetail;
  detailProblem?: 'pending' | 'db_unreachable';
}

interface PanelsAnalytics {
  range: RangeKey;
  step: number;
  panels: PanelStats[];
}

// Polling is marked as background, so an open page does not keep the session alive.
const polling = { background: true } as const;

export function getServerStats(range: RangeKey, after?: number): Promise<ServerStats> {
  const q = new URLSearchParams({ range });
  if (after) q.set('after', String(after));
  return api<ServerStats>('GET', `analytics/server?${q.toString()}`, undefined, polling);
}

export function getPanelsStats(range: RangeKey): Promise<PanelsAnalytics> {
  return api<PanelsAnalytics>('GET', `analytics/panels?range=${range}`, undefined, polling);
}

export function getPanelStats(id: number, range: RangeKey): Promise<PanelAnalytics> {
  return api<PanelAnalytics>('GET', `analytics/panels/${id}?range=${range}`, undefined, polling);
}
