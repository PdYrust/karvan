import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { SignOutIcon, UserCircleIcon } from '@phosphor-icons/react';
import { locales } from '../../i18n/core';
import { useLocale, useT } from '../../i18n/locale';
import { languageNames, sharedMessages } from '../../i18n/shared';
import { shellMessages } from './messages';
import { useSession, useSignOut } from './use-session';

export function AccountMenu() {
  const t = useT(shellMessages);
  const ts = useT(sharedMessages);
  const { locale, setLocale } = useLocale();
  const session = useSession();
  const signOut = useSignOut();
  const username = session.data?.operator?.username ?? '';
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={t('account')}
            title={t('account')}
            className="pointer-coarse:size-11"
            icon={<UserCircleIcon size={22} />}
          />
        }
      />
      <DropdownMenu.Content className="min-w-56">
        <DropdownMenu.Group>
          <DropdownMenu.Label>
            <span className="block text-tiny text-kumo-subtle">{t('signedInAs')}</span>
            <bdi dir="ltr" className="font-mono text-sm text-kumo-default">
              {username}
            </bdi>
          </DropdownMenu.Label>
        </DropdownMenu.Group>
        <DropdownMenu.Separator />
        <DropdownMenu.Group>
          <DropdownMenu.Label>{ts('languageLabel')}</DropdownMenu.Label>
          <DropdownMenu.RadioGroup
            value={locale}
            onValueChange={(v) => setLocale(v as typeof locale)}
          >
            {locales.map((l) => (
              <DropdownMenu.RadioItem key={l} value={l} lang={l}>
                <span className="flex w-full items-center gap-2">
                  {languageNames[l]}
                  <span className="ms-auto">
                    <DropdownMenu.RadioItemIndicator />
                  </span>
                </span>
              </DropdownMenu.RadioItem>
            ))}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Group>
        <DropdownMenu.Separator />
        <DropdownMenu.Item
          icon={<SignOutIcon size={16} className="rtl:-scale-x-100" />}
          onClick={signOut}
        >
          {t('signOut')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
