import { Button } from '@cloudflare/kumo/components/button';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { SettingRow, SettingsCard } from '../../../components/layout/SettingRow';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import { accountKeys, renewLogin, type NewLogin } from '../../../lib/api/account';
import type { SessionState } from '../../../lib/api/session';
import { sessionKey, useSession } from '../../session/use-session';
import { ReauthDialog } from '../security/ReauthDialog';
import { ShownOnce } from '../security/ShownOnce';
import { accessMessages } from './messages';

/** The operator's username, and a new username and password with one click, shown once. */
export function LoginCard() {
  const t = useT(accessMessages);
  const session = useSession();
  const queryClient = useQueryClient();
  const [asking, setAsking] = useState(false);
  // One-time values (D10): component state only, gone on "I saved them" or unmount.
  const [made, setMade] = useState<NewLogin | null>(null);
  const renew = useMutation({
    mutationFn: renewLogin,
    gcTime: 0,
    onSuccess: (login) => {
      setAsking(false);
      setMade(login);
      queryClient.setQueryData<SessionState>(sessionKey, (s) =>
        s?.operator ? { ...s, operator: { ...s.operator, username: login.username } } : s,
      );
      void queryClient.invalidateQueries({ queryKey: accountKeys.sessions });
    },
  });
  const name = session.data?.operator?.username ?? '';

  return (
    <>
      {made ? (
        <ShownOnce
          title={t('loginShownTitle')}
          values={[
            { label: t('username'), value: made.username },
            { label: t('password'), value: made.password },
          ]}
          onDone={() => setMade(null)}
        >
          <p className="text-kumo-default">{t('loginShownNote')}</p>
        </ShownOnce>
      ) : null}
      <SettingsCard title={t('loginTitle')}>
        <SettingRow
          title={t('usernameTitle')}
          description={name ? <CodeText text={t('usernameHint', { name: `\`${name}\`` })} /> : null}
          control={
            <Button
              className="pointer-coarse:h-9"
              onClick={() => {
                renew.reset();
                setAsking(true);
              }}
            >
              {t('newLogin')}
            </Button>
          }
        />
      </SettingsCard>
      <ReauthDialog
        open={asking}
        onOpenChange={setAsking}
        title={t('loginDialogTitle')}
        description={t('loginDialogBody')}
        action={t('makeLogin')}
        pending={renew.isPending}
        error={renew.error}
        onSubmit={(r) => renew.mutate(r)}
      />
    </>
  );
}
