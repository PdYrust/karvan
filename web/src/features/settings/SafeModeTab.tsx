import { Button } from '@cloudflare/kumo/components/button';
import { Radio } from '@cloudflare/kumo/components/radio';
import { Switch } from '@cloudflare/kumo/components/switch';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useState } from 'react';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { SettingRow, SettingsCard } from '../../components/layout/SettingRow';
import { useLocale, useT } from '../../i18n/locale';
import { turnOffConfirm, type Rollback, type SafeMode } from '../../lib/api/safemode';
import { usePanels } from '../panels/use-panels';
import { settingsMessages } from './messages';
import { useSafeMode, useSetSafeMode } from './use-safe-mode';
import { Reveal } from '../../components/feedback/Reveal';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { switchSlide, useMinDuration } from '../../lib/motion/use-min-duration';

function Choice({ title, hint }: { title: string; hint: string }) {
  return (
    <span className="flex flex-col gap-1">
      <span className="font-medium text-kumo-strong">{title}</span>
      <span className="text-sm text-kumo-subtle">{hint}</span>
    </span>
  );
}

export function SafeModeTab() {
  const t = useT(settingsMessages);
  const { locale } = useLocale();
  const safe = useSafeMode();
  const set = useSetSafeMode();
  const panels = usePanels();
  const toasts = useKumoToastManager();
  const [confirming, setConfirming] = useState(false);
  // Busy until the thumb has slid, however fast the save answers.
  const saving = useMinDuration(set.isPending, switchSlide);

  const load = useFirstLoad(safe.data);
  if (safe.isPending || load.holding) {
    return (
      <div role="status" aria-label={t('safeModeTitle')} className="flex flex-col gap-6">
        <Bone className="h-28 w-full" />
        <Bone className="h-48 w-full" />
      </div>
    );
  }
  if (safe.isError) {
    return (
      <div className="flex flex-col items-start gap-3">
        <InlineAlert>{t('loadFailed')}</InlineAlert>
        <Button onClick={() => void safe.refetch()}>{t('retry')}</Button>
      </div>
    );
  }
  const s = safe.data;
  const noControl = (panels.data ?? [])
    .filter((p) => p.id > 0 && p.control && !p.control.available)
    .map((p) => p.name);

  function save(next: SafeMode, done: string, confirm?: string) {
    set.mutate(
      { next, confirm },
      {
        onSuccess: () => {
          setConfirming(false);
          toasts.add({ title: done });
        },
        onError: () => toasts.add({ title: t('saveFailed'), variant: 'error' }),
      },
    );
  }

  return (
    <Reveal on={load.reveal}>
      <SettingsCard>
        <SettingRow
          title={t('safeModeTitle')}
          labelId="safe-mode-title"
          description={t('safeModeHint')}
          inline
          control={
            <Switch
              aria-labelledby="safe-mode-title"
              // The thumb moves on the press; a failed save slides it back.
              checked={set.isPending ? set.variables.next.enabled : s.enabled}
              disabled={saving}
              // The switch is 18 px high; the hit area reaches 44 px on touch screens.
              className="relative pointer-coarse:before:absolute pointer-coarse:before:-inset-x-[9px] pointer-coarse:before:-inset-y-[13px] pointer-coarse:before:content-['']"
              onCheckedChange={(on) =>
                on ? save({ ...s, enabled: true }, t('turnedOn')) : setConfirming(true)
              }
            />
          }
        >
          {s.enabled ? null : (
            <InlineAlert tone="warning" className="mt-2">
              {t('safeModeOffWarning')}
            </InlineAlert>
          )}
        </SettingRow>
      </SettingsCard>

      <SettingsCard>
        <div className="flex flex-col gap-4 py-5">
          <Radio.Group
            legend={t('rollbackTitle')}
            description={t('rollbackHint')}
            appearance="card"
            value={s.rollback}
            disabled={saving}
            onValueChange={(v: Rollback) => save({ ...s, rollback: v }, t('rollbackSaved'))}
          >
            <Radio.Item
              value="undo"
              label={<Choice title={t('rollbackUndo')} hint={t('rollbackUndoHint')} />}
            />
            <Radio.Item
              value="backup"
              label={<Choice title={t('rollbackBackup')} hint={t('rollbackBackupHint')} />}
            />
          </Radio.Group>
          {s.rollback === 'backup' && noControl.length > 0 ? (
            <InlineAlert tone="warning">
              {t('controlMissing', {
                count: noControl.length,
                names: new Intl.ListFormat(locale).format(noControl),
              })}
            </InlineAlert>
          ) : null}
        </div>
      </SettingsCard>

      <TypeToConfirm
        open={confirming}
        onOpenChange={setConfirming}
        title={t('turnOffTitle')}
        word={turnOffConfirm}
        action={t('turnOffAction')}
        pending={set.isPending}
        onConfirm={() => save({ ...s, enabled: false }, t('turnedOff'), turnOffConfirm)}
      >
        {t('turnOffBody')}
      </TypeToConfirm>
    </Reveal>
  );
}
