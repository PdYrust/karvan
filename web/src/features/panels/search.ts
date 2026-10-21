import { panelSearchNames } from '../../components/panels/PanelLogo';
import { databaseOf, engineName, panelKind, panelRelease, type Panel } from '../../lib/api/panels';

/** Whether a panel matches the search field: its name, folder, kind, release or database. */
export function matchesPanel(p: Panel, query: string): boolean {
  const q = query.trim().toLocaleLowerCase();
  if (!q) return true;
  const kind = panelKind(p);
  const { engine } = databaseOf(p);
  return [
    p.name,
    p.dir,
    p.db,
    kind ? panelSearchNames[kind] : undefined,
    panelRelease(p.revision),
    engine ? engineName(engine) : undefined,
  ].some((v) => v?.toLocaleLowerCase().includes(q));
}
