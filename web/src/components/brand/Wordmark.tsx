import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { Mark } from './Mark';

/** Mark and name together, sized for headers. */
export function Wordmark({ size = 28 }: { size?: number }) {
  const t = useT(sharedMessages);
  return (
    <span className="inline-flex items-center gap-3">
      <Mark size={size} />
      <span className="text-heading font-extrabold tracking-tight text-kumo-strong">
        {t('brand')}
      </span>
    </span>
  );
}
