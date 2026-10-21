import { Button } from '@cloudflare/kumo/components/button';
import { ArrowsClockwiseIcon, HardDrivesIcon } from '@phosphor-icons/react';
import { EmptyState } from '../../components/feedback/EmptyState';
import { PanelMark } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { panelsMessages } from './messages';

/** No panel registered yet: the two panels Karvan works with, and the scan. */
export function NoPanels({ onScan, scanning }: { onScan: () => void; scanning: boolean }) {
  const t = useT(panelsMessages);
  return (
    <EmptyState
      icon={<HardDrivesIcon size={44} weight="light" aria-hidden="true" />}
      title={t('emptyTitle')}
      action={
        <Button
          icon={<ArrowsClockwiseIcon size={16} />}
          loading={scanning}
          onClick={onScan}
          className="pointer-coarse:h-11"
        >
          {t('scan')}
        </Button>
      }
    >
      <p>
        <CodeText text={t('emptyBody')} />
      </p>
      <ul
        aria-label={t('supported')}
        className="mt-4 flex items-center justify-center gap-5 text-sm text-kumo-default"
      >
        {(['marzban', 'pasarguard'] as const).map((k) => (
          <li key={k} className="flex items-center gap-2">
            <PanelMark kind={k} size={18} />
            {panelNames[k]}
          </li>
        ))}
      </ul>
    </EmptyState>
  );
}
