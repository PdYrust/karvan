import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { Mark } from './Mark';

/** Mark and name together, sized for headers. */
export function Wordmark({ size = 26 }: { size?: number }) {
  const t = useT(sharedMessages);
  return (
    <span className="inline-flex items-center gap-2">
      <Mark size={size} />
      <span className="text-[15px] font-semibold text-kumo-strong">{t('brand')}</span>
    </span>
  );
}
