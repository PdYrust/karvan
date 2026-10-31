import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { LocaleProvider } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { PanelCardFoot } from './PanelsTable';
// React's act() needs this flag outside a testing library.
(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  document.documentElement.lang = 'en';
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
  document.documentElement.lang = '';
});
function foot(over: Partial<Panel>): string {
  const panel = { id: 1, name: 'pasarguard', findings: [], ready: true, ...over } as Panel;
  act(() =>
    root.render(
      <LocaleProvider>
        <PanelCardFoot panel={panel} />
      </LocaleProvider>,
    ),
  );
  return host.textContent ?? '';
}
describe('PanelCardFoot', () => {
  it('shows only Ready when the panel is ready and panel control is available', () => {
    expect(foot({ control: { available: true } })).toBe('Ready');
    expect(host.querySelectorAll('svg')).toHaveLength(1);
  });
  it('adds one warning when panel control is missing', () => {
    expect(foot({ control: { available: false, reason: 'docker_denied' } })).toBe(
      'ReadyPanel control: No Docker access',
    );
  });
  it('keeps the not-ready state', () => {
    const text = foot({
      ready: false,
      findings: [{ code: 'schema_mismatch' }] as Panel['findings'],
      control: { available: true },
    });
    expect(text).toBe('1 problem');
  });
});
