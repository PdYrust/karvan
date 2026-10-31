import { describe, expect, it } from 'vitest';
import type { CertCopy, Certificate } from '../../lib/api/certificates';
import type { Panel } from '../../lib/api/panels';
import { copyChoices, isRealPanel } from './copy-targets';

const panel = (id: number, name: string, over: Partial<Panel> = {}) =>
  ({ id, name, kind: 'marzban', findings: [], ...over }) as Panel;

const copy = (over: Partial<CertCopy>): CertCopy => ({
  id: 1,
  target: 'letsencrypt',
  panel: null,
  dir: '/etc/letsencrypt/live/a.example.com',
  certFile: '',
  keyFile: '',
  copiedAt: null,
  error: null,
  ...over,
});

const cert = (copies: CertCopy[]) => ({ id: 4, copies }) as unknown as Certificate;

describe('the Copy to a panel picker', () => {
  it("lists Let's Encrypt live first, then the real panels by name", () => {
    const rows = copyChoices(cert([]), [
      panel(2, 'pasarguard', { kind: 'pasarguard' }),
      panel(1, 'marzban'),
      panel(0, 'not-registered'),
      panel(3, 'unknown', { kind: undefined }),
      panel(4, 'marzban-old', { findings: [{ code: 'panel_copy', detail: '/opt/marzban' }] }),
    ]);
    expect(rows.map((r) => r.key)).toEqual(['letsencrypt', 'panel:1', 'panel:2']);
    expect(rows[1]).toEqual({
      key: 'panel:1',
      target: { target: 'panel', panelId: 1 },
      panel: { name: 'marzban', kind: 'marzban' },
      copied: false,
    });
  });

  it('checks the places it is copied to already', () => {
    const rows = copyChoices(
      cert([
        copy({}),
        copy({ id: 2, target: 'panel', panel: { id: 2, name: 'pasarguard', kind: 'pasarguard' } }),
      ]),
      [panel(1, 'marzban'), panel(2, 'pasarguard', { kind: 'pasarguard' })],
    );
    expect(rows.map((r) => r.copied)).toEqual([true, false, true]);
  });

  it('takes only registered panels of a known kind that are not copies', () => {
    expect(isRealPanel(panel(1, 'a'))).toBe(true);
    expect(isRealPanel(panel(0, 'a'))).toBe(false);
    expect(isRealPanel(panel(1, 'a', { kind: undefined }))).toBe(false);
    expect(isRealPanel(panel(1, 'a', { findings: [{ code: 'panel_copy' }] }))).toBe(false);
    expect(isRealPanel(panel(1, 'a', { findings: [{ code: 'db_unreachable' }] }))).toBe(true);
  });
});
