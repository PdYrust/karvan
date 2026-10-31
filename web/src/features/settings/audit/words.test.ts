import { describe, expect, it } from 'vitest';
import { actionMessages, auditActions, isAuditAction, isAuditReason } from './words';

// The actions internal/audit/audit.go defines.
const serverActions = [
  'login',
  'logout',
  'lockout',
  'totp.enable',
  'totp.disable',
  'cli.init',
  'cli.reset',
  'panels.scan',
  'panels.copy_removed',
  'migration.run',
  'migration.rollback',
  'migration.cancel',
  'migration.delete',
  'migration.passwords_download',
  'safe_mode.on',
  'safe_mode.off',
  'safe_mode.rollback',
  'safe_mode.backup_delete',
  'safe_mode.backup_restore',
  'cli.safe_mode_restore',
  'legacy_site.save',
  'legacy_site.delete',
  'legacy_site.tokens',
  'batch.rename',
  'batch.delete',
  'batch.reactivate',
  'batch.show_link',
  'domain.add',
  'domain.delete',
  'domain.certificate',
  'certificate.add',
  'certificate.save',
  'certificate.delete',
  'certificate.scan',
  'redirect.save',
  'redirect.delete',
  'sessions.end',
  'account.credentials',
  'settings.address_start',
  'settings.address',
  'settings.https',
];

describe('audit words', () => {
  it('names every action the server records, in both languages', () => {
    expect([...auditActions].sort()).toEqual([...serverActions].sort());
    for (const a of auditActions) {
      expect(actionMessages.en[a], a).toBeTruthy();
      expect(actionMessages.fa[a], a).toBeTruthy();
    }
  });

  it('does not take inherited keys or new actions for known ones', () => {
    expect(isAuditAction('constructor')).toBe(false);
    expect(isAuditAction('panels.delete')).toBe(false);
    expect(isAuditReason('toString')).toBe(false);
    expect(isAuditReason(undefined)).toBe(false);
    expect(isAuditReason('bad_totp')).toBe(true);
  });
});
