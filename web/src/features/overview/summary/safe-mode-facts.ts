import type { Translate } from '../../../i18n/core';
import type { SafeMode } from '../../../lib/api/safemode';
import type { OverviewKey } from '../messages';

/**
 * What a failed migration does to the new panel. With Safe Mode off a run
 * takes no backup and undoes nothing, so the saved choice shows only as
 * what happens with Safe Mode on.
 */
export function failureFacts(
  safe: SafeMode,
  t: Translate<OverviewKey>,
): { label: string; value: string }[] {
  const choice = t(safe.rollback === 'backup' ? 'rollbackBackup' : 'rollbackUndo');
  if (safe.enabled) return [{ label: t('ifFails'), value: choice }];
  return [
    { label: t('ifFails'), value: t('rollbackOff') },
    { label: t('withSafeMode'), value: choice },
  ];
}
