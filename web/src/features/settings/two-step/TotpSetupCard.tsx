import { Button } from '@cloudflare/kumo/components/button';
import { useMutation } from '@tanstack/react-query';
import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from 'react';
import { dialogMessages } from '../../../components/dialogs/messages';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { CopyButton } from '../../../components/text/CopyButton';
import { useT } from '../../../i18n/locale';
import { confirmTotp, type TotpSetup } from '../../../lib/api/account';
import { useErrorText } from '../../../lib/errors/errors';
import { useFormat } from '../../../lib/preferences/region';
import { groupSecret } from '../security/code';
import { CodeField } from '../security/CodeField';
import { twoStepMessages } from './messages';

// The encoder loads when setup starts, so the Settings page does not carry it.
export const loadQr = () => import('../../../lib/qr/qr-path');

/** The QR code, dark on a white tile of a fixed size in both themes so every camera reads it. */
function QrCode({ text, label }: { text: string; label: string }) {
  const [qr, setQr] = useState<{ size: number; d: string } | null>(null);
  useEffect(() => {
    let live = true;
    void loadQr().then((m) => {
      if (live) setQr(m.qrPath(text));
    });
    return () => {
      live = false;
    };
  }, [text]);
  if (!qr) return <Bone className="size-52 shrink-0 rounded-lg" />;
  return (
    <svg
      role="img"
      aria-label={label}
      viewBox={`0 0 ${String(qr.size)} ${String(qr.size)}`}
      shapeRendering="crispEdges"
      className="k-fade size-52 shrink-0 rounded-lg bg-qr-paper ring ring-kumo-hairline"
    >
      <path d={qr.d} className="fill-qr-ink" />
    </svg>
  );
}

/** The secret in groups of four, in mono, wrapping under the code, with Copy. */
function SecretKey({ secret }: { secret: string }) {
  const t = useT(twoStepMessages);
  const box = useRef<HTMLParagraphElement>(null);
  // Where the browser copies nothing, the key is selected to copy by hand.
  const select = () => {
    if (box.current) window.getSelection()?.selectAllChildren(box.current);
  };
  return (
    <div className="flex w-52 flex-col items-stretch gap-2">
      <p className="text-center text-sm text-kumo-subtle">{t('key')}</p>
      <p
        ref={box}
        dir="ltr"
        className="rounded-lg bg-kumo-tint px-3 py-2 text-center font-mono text-sm leading-6 break-words text-kumo-strong select-all"
      >
        {groupSecret(secret)}
      </p>
      <CopyButton text={secret} select={select} className="w-full pointer-coarse:h-9" />
    </div>
  );
}

/** One numbered step: its number, one short sentence, then what to do. */
function Step({ n, text, children }: { n: number; text: string; children: ReactNode }) {
  const fmt = useFormat();
  return (
    <li className="flex min-w-0 flex-col gap-4">
      <div className="flex items-start gap-3">
        <span
          aria-hidden="true"
          className="grid size-6 shrink-0 place-items-center rounded-full bg-kumo-tint text-sm font-medium text-kumo-strong tabular"
        >
          {fmt.number(n)}
        </span>
        <p className="pt-0.5 text-kumo-default">{text}</p>
      </div>
      {children}
    </li>
  );
}

/** Under the setting row: 1, scan the code or type the key; 2, confirm with the code the app shows. */
export function TotpSetupCard({
  setup,
  onDone,
  onCancel,
}: {
  setup: TotpSetup;
  onDone: () => void;
  onCancel: () => void;
}) {
  const t = useT(twoStepMessages);
  const td = useT(dialogMessages);
  const errorText = useErrorText();
  const id = useId();
  const [code, setCode] = useState('');
  const confirm = useMutation({ mutationFn: confirmTotp, gcTime: 0, onSuccess: onDone });
  function submit(e: FormEvent) {
    e.preventDefault();
    if (code.length === 6 && !confirm.isPending) confirm.mutate(code);
  }
  return (
    <form
      aria-label={t('setupTitle')}
      onSubmit={submit}
      className="k-fade flex flex-col gap-6 border-b border-kumo-hairline py-6 last:border-b-0"
    >
      <ol className="grid gap-8 md:grid-cols-2">
        <Step n={1} text={t('scan')}>
          <div className="flex flex-col items-center gap-4">
            <QrCode text={setup.uri} label={t('qrLabel')} />
            <SecretKey secret={setup.secret} />
          </div>
        </Step>
        <Step n={2} text={t('enterCode')}>
          <div className="flex flex-col gap-4 sm:ps-9">
            <CodeField
              id={`${id}-code`}
              label={t('enterCode')}
              hideLabel
              value={code}
              onChange={setCode}
              disabled={confirm.isPending}
            />
            {confirm.isError ? <InlineAlert>{errorText(confirm.error)}</InlineAlert> : null}
          </div>
        </Step>
      </ol>
      <div className="flex justify-end gap-2 border-t border-kumo-hairline pt-4">
        <Button
          variant="secondary"
          className="pointer-coarse:h-9"
          disabled={confirm.isPending}
          onClick={onCancel}
        >
          {td('cancel')}
        </Button>
        <Button
          type="submit"
          variant="primary"
          className="pointer-coarse:h-9"
          disabled={code.length !== 6}
          loading={confirm.isPending}
        >
          {t('turnOn')}
        </Button>
      </div>
    </form>
  );
}
