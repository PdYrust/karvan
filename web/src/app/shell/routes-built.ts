import type { ShellKey } from './messages';
import { nav, settingsPath, type NavItem } from './nav';

const items: NavItem[] = nav.flatMap((g) => g.items);

/** The route of a sidebar page. */
export function pathOf(label: ShellKey): string | undefined {
  return items.find((i) => i.label === label)?.to;
}

/** Physical key code after `g` to the route it opens. */
export function gotoTargets(): Record<string, string> {
  const out: Record<string, string> = { KeyS: settingsPath };
  for (const i of items) out[i.key] = i.to;
  return out;
}

/** Every sidebar page with its shortcut letter, for the shortcut sheet and the palette. */
export function builtPages(): NavItem[] {
  return items;
}
