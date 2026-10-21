import type { EngineStats } from '../../../lib/api/analytics';
import { engineName } from '../../../lib/api/panels';

/** "SQLite 3.53.4", "PostgreSQL 17.2". Engine names stay in Latin in both languages. */
export function engineLabel(e: EngineStats | undefined): string {
  if (!e) return '';
  return `${engineName(e.flavor)} ${e.version}`.trim();
}

/** SQLite keeps no query or cache counters and has no server to connect to. */
export function isServerEngine(e: EngineStats | undefined): boolean {
  return e !== undefined && e.flavor !== 'sqlite';
}
