import { Button } from '@cloudflare/kumo/components/button';
import { useMutation } from '@tanstack/react-query';
import { useId, useMemo, useState, type FormEvent } from 'react';
import { dialogMessages } from '../../../components/dialogs/messages';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { useT } from '../../../i18n/locale';
import { confirmTotp, type TotpSetup } from '../../../lib/api/account';
import { useErrorText } from '../../../lib/errors/errors';
import { groupSecret } from '../security/code';
import { CodeField } from '../security/CodeField';
import { CopyValue } from '../security/CopyValue';
import { twoStepMessages } from './messages';
import { qrPath } from './qr-path';

/** The QR code, drawn dark on a white tile in both themes so every camera reads it. */
function QrCode({ text, label }: { text: string; label: string }) {
  const { size, d } = useMemo(() => qrPath(text), [text]);
  return (
    <svg
      role="img"
      aria-label={label}
      viewBox={`0 0 ${size} ${size}`}
      shapeRendering="crispEdges"
      className="size-48 shrink-0 rounded-lg bg-qr-paper ring ring-kumo-hairline"
    >
      <path d={d} className="fill-qr-ink" />
    </svg>
  );
}

/** Scan the code or type the key, then confirm with the first code the app shows. */
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
    <section
      aria-labelledby={id}
      className="flex flex-col gap-5 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-6"
    >
      <h2 id={id} className="text-heading font-semibold text-kumo-strong">
        {t('setupTitle')}
      </h2>
      <div className="flex flex-col gap-5 sm:flex-row sm:items-start sm:gap-8">
        <QrCode text={setup.uri} label={t('qrLabel')} />
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          <p className="text-kumo-default">{t('scan')}</p>
          <CopyValue label={t('key')} value={setup.secret} shown={groupSecret(setup.secret)} />
        </div>
      </div>
      <form onSubmit={submit} className="flex flex-col gap-4 border-t border-kumo-hairline pt-5">
        <CodeField
          id={`${id}-code`}
          label={t('enterCode')}
          value={code}
          onChange={setCode}
          disabled={confirm.isPending}
        />
        {confirm.isError ? <InlineAlert>{errorText(confirm.error)}</InlineAlert> : null}
        <div className="flex justify-end gap-2">
          <Button
            variant="secondary"
            className="pointer-coarse:h-11"
            disabled={confirm.isPending}
            onClick={onCancel}
          >
            {td('cancel')}
          </Button>
          <Button
            type="submit"
            variant="primary"
            className="pointer-coarse:h-11"
            disabled={code.length !== 6}
            loading={confirm.isPending}
          >
            {t('turnOn')}
          </Button>
        </div>
      </form>
    </section>
  );
}
