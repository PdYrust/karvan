import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { DotsThreeIcon, LinkSimpleIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { copyText } from '../../lib/clipboard/copy-text';
import { analyticsMessages } from './messages';

/** The "..." at the end of the analytics toolbar: a link to this tab and range. */
export function PageMenu() {
  const t = useT(analyticsMessages);
  const toasts = useKumoToastManager();
  const copy = () =>
    void copyText(window.location.href).then((ok) =>
      toasts.add({ title: t(ok ? 'linkCopied' : 'linkNotCopied') }),
    );
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            className="shrink-0 pointer-coarse:size-9"
            aria-label={t('pageMenu')}
            title={t('pageMenu')}
            icon={<DotsThreeIcon size={18} weight="bold" />}
          />
        }
      />
      <DropdownMenu.Content align="end" className="min-w-48">
        <DropdownMenu.Item onClick={copy}>
          <span className="flex items-center gap-2">
            <LinkSimpleIcon size={16} aria-hidden="true" />
            {t('copyLink')}
          </span>
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
