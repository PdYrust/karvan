import type { ShellKey } from './messages';
import { nav, settingsPath, type NavItem } from './nav';

const items: NavItem[] = nav.flatMap((g) => g.items);

/** The route of a sidebar page, or undefined while it is not built yet. */
export function pathOf(label: ShellKey): string | undefined {
  return items.find((i) => i.label === label)?.to;
}

/** Physical key code after `g` to the route it opens. */
export function gotoTargets(): Record<string, string> {
  const out: Record<string, string> = { KeyS: settingsPath };
  for (const i of items) {
    if (i.to && i.key) out[i.key] = i.to;
  }
  return out;
}

/** Built pages with their shortcut letters, for the shortcut sheet. */
export function builtPages(): NavItem[] {
  return items.filter((i) => i.to);
}
