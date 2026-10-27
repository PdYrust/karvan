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
  if (!qr) return <Bone className="size-48 shrink-0 rounded-lg sm:size-52" />;
  return (
    <svg
      role="img"
      aria-label={label}
      viewBox={`0 0 ${String(qr.size)} ${String(qr.size)}`}
      shapeRendering="crispEdges"
      className="k-fade size-48 shrink-0 rounded-lg sm:size-52 bg-qr-paper ring ring-kumo-hairline"
    >
      <path d={qr.d} className="fill-qr-ink" />
    </svg>
  );
}

/** The secret in groups of four, in mono, wrapping under the code, with Copy next to it. */
function SecretKey({ secret }: { secret: string }) {
  const t = useT(twoStepMessages);
  const box = useRef<HTMLParagraphElement>(null);
  // Where the browser copies nothing, the key is selected to copy by hand.
  const select = () => {
    if (box.current) window.getSelection()?.selectAllChildren(box.current);
  };
  return (
    <div className="flex w-full max-w-sm flex-col gap-2">
      <p className="text-sm text-kumo-subtle">{t('key')}</p>
      <div className="flex items-start gap-2">
        <p
          ref={box}
          dir="ltr"
          className="min-w-0 flex-1 rounded-lg bg-kumo-tint px-3 py-1 font-mono text-sm leading-6 break-words text-kumo-strong select-all"
        >
          {groupSecret(secret)}
        </p>
        <CopyButton text={secret} select={select} />
      </div>
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
          className="grid size-7 shrink-0 place-items-center rounded-full bg-kumo-base text-sm font-semibold text-kumo-strong ring ring-kumo-line tabular"
        >
          {fmt.number(n)}
        </span>
        <p className="pt-1 text-kumo-default">{text}</p>
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
    <form aria-label={t('setupTitle')} onSubmit={submit} className="flex flex-col gap-6 py-6">
      {/* Stacked on phones with a hairline between the steps, side by side from 768 px. */}
      <ol className="grid gap-6 md:grid-cols-2 md:gap-8 max-md:[&>li+li]:border-t max-md:[&>li+li]:border-kumo-hairline max-md:[&>li+li]:pt-6">
        <Step n={1} text={t('scan')}>
          <div className="flex flex-col items-center gap-4">
            <QrCode text={setup.uri} label={t('qrLabel')} />
            <SecretKey secret={setup.secret} />
          </div>
        </Step>
        <Step n={2} text={t('enterCode')}>
          <div className="flex flex-col gap-4 sm:ps-10">
            <CodeField
              id={`${id}-code`}
              label={t('enterCode')}
              hideLabel
              value={code}
              onChange={setCode}
              disabled={confirm.isPending}
              fill
            />
            {confirm.isError ? <InlineAlert>{errorText(confirm.error)}</InlineAlert> : null}
          </div>
        </Step>
      </ol>
      {/* On phones both span the width, Turn on last, at the bottom. */}
      <div className="flex flex-col gap-2 border-t border-kumo-hairline pt-4 sm:flex-row sm:justify-end">
        <Button
          variant="secondary"
          className="w-full sm:w-auto pointer-coarse:h-9"
          disabled={confirm.isPending}
          onClick={onCancel}
        >
          {td('cancel')}
        </Button>
        <Button
          type="submit"
          variant="primary"
          className="w-full sm:w-auto pointer-coarse:h-9"
          disabled={code.length !== 6}
          loading={confirm.isPending}
        >
          {t('turnOn')}
        </Button>
      </div>
    </form>
  );
}
