import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { CopyIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { copyText } from '../../lib/clipboard/copy-text';
import { textMessages } from './messages';

/**
 * Copy, with a toast that says it worked. Where the browser copies nothing,
 * `select` readies the text to copy by hand and the toast says so.
 */
export function CopyButton({
  text,
  select,
  disabled,
  className = 'shrink-0 pointer-coarse:h-11',
}: {
  text: string;
  select?: () => void;
  disabled?: boolean;
  className?: string;
}) {
  const t = useT(textMessages);
  const toasts = useKumoToastManager();
  const copy = () =>
    void copyText(text).then((ok) => {
      if (!ok) select?.();
      toasts.add({ title: t(ok ? 'copied' : 'copyBlocked') });
    });
  return (
    <Button className={className} disabled={disabled} icon={<CopyIcon size={16} />} onClick={copy}>
      {t('copy')}
    </Button>
  );
}
