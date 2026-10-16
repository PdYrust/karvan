import type { PanelKind } from '../../components/panels/PanelLogo';

/**
 * What the globe on the sign-in page acts out, in turn. A move sends every user
 * and the old panel goes away; a merge adds users and both panels stay.
 */
export interface Scene {
  from: PanelKind;
  to: PanelKind;
  kind: 'move' | 'merge';
}

export const scenes: readonly [Scene, ...Scene[]] = [
  { from: 'marzban', to: 'pasarguard', kind: 'move' },
  { from: 'pasarguard', to: 'marzban', kind: 'merge' },
  { from: 'pasarguard', to: 'marzban', kind: 'move' },
  { from: 'marzban', to: 'pasarguard', kind: 'merge' },
];

/** The scene for a running count; scenes repeat in order. */
export const sceneAt = (index: number): Scene => scenes[index % scenes.length] ?? scenes[0];

/** Milliseconds per scene. */
export const sceneLength = 6500;
