import { Button, LinkButton } from '@cloudflare/kumo/components/button';
import { ArrowSquareOutIcon } from '@phosphor-icons/react';
import { useId } from 'react';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { useT } from '../../../i18n/locale';
import type { PendingAddress } from '../../../lib/api/account';
import { useFormat } from '../../../lib/preferences/region';
import { CopyValue } from '../security/CopyValue';
import { accessMessages } from './messages';
import type { WaitingLink } from './waiting';

/**
 * A new address Karvan serves next to the old one: until when, the link
 * that opens it in a new tab, and Confirm once the page came through it.
 */
export function WaitingCard({
  pending,
  link,
  secondsLeft,
  confirming,
  confirmError,
  onConfirm,
}: {
  pending: PendingAddress;
  link: WaitingLink;
  secondsLeft: number;
  confirming: boolean;
  confirmError: string | null;
  onConfirm: () => void;
}) {
  const t = useT(accessMessages);
  const fmt = useFormat();
  const id = useId();
  const left = (
    <span className="text-sm text-kumo-subtle tabular" aria-live="off">
      {t('timeLeft', { time: fmt.clock(secondsLeft) })}
    </span>
  );
  return (
    <section
      aria-labelledby={id}
      className="flex flex-col gap-4 rounded-lg bg-kumo-base p-4 ring ring-kumo-info/40 sm:p-6"
    >
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id={id} className="text-heading font-semibold text-kumo-strong">
          {t(pending.fromNew ? 'confirmTitle' : 'waitingTitle')}
        </h2>
        {left}
      </div>
      {pending.fromNew ? (
        <>
          <p className="max-w-prose text-kumo-default">{t('confirmBody')}</p>
          {confirmError ? <InlineAlert>{confirmError}</InlineAlert> : null}
          <div className="flex justify-end">
            <Button
              variant="primary"
              className="pointer-coarse:h-11"
              loading={confirming}
              onClick={onConfirm}
            >
              {t('confirm')}
            </Button>
          </div>
        </>
      ) : (
        <>
          <p className="max-w-prose text-kumo-default">
            {t('waitingBody', { time: fmt.time(new Date(pending.expiresAt)) })}
          </p>
          {link.url ? (
            <>
              <CopyValue label={t('newAddress')} value={link.url} />
              {link.signInFirst ? <InlineAlert tone="info">{t('signInThere')}</InlineAlert> : null}
              <div className="flex justify-end">
                <LinkButton
                  href={link.url}
                  external
                  variant="primary"
                  className="gap-2 pointer-coarse:h-11"
                >
                  {t('openNew')}
                  <ArrowSquareOutIcon size={16} className="rtl:-scale-x-100" />
                </LinkButton>
              </div>
            </>
          ) : (
            <InlineAlert tone="info">{t('pathShownOnce')}</InlineAlert>
          )}
        </>
      )}
    </section>
  );
}
