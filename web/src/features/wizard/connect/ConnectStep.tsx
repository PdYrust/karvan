import { Button } from '@cloudflare/kumo/components/button';
import type { FormEvent } from 'react';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { useT } from '../../../i18n/locale';
import { useErrorText } from '../../../lib/errors/errors';
import { wizardMessages } from '../messages';
import { WizardFrame } from '../WizardFrame';
import { CertificateCard } from './CertificateCard';
import { DestinationField } from './DestinationField';
import { connectMessages, type ConnectKey } from './messages';
import { ReadProgress } from './ReadProgress';
import { SourceFields } from './SourceFields';
import { useConnect } from './use-connect';

type Connect = ReturnType<typeof useConnect>;

/** A read that stopped: why, the certificate to trust when that is the reason, and the ways on. */
function ReadStopped({ c }: { c: Connect }) {
  const t = useT(connectMessages);
  const errorText = useErrorText();
  const reason = c.stream.failure ?? c.compare.error;
  // A read the operator stopped needs no "couldn't finish" in front of it.
  const stopped = c.stream.failure?.code === 'cancelled';
  return (
    <div className="flex max-w-[720px] flex-col gap-4">
      {c.cert ? (
        <CertificateCard
          cert={c.cert}
          pending={c.create.isPending}
          onTrust={() => c.cert && c.trust(c.cert)}
        />
      ) : (
        <InlineAlert tone={stopped ? 'info' : 'danger'}>
          {stopped ? errorText(reason) : `${t('readFailed')} ${errorText(reason)}`}
        </InlineAlert>
      )}
      <div className="flex flex-wrap justify-end gap-2">
        <Button className="pointer-coarse:h-11" onClick={c.changeDetails}>
          {t('changeDetails')}
        </Button>
        {c.cert ? null : (
          <Button
            variant="primary"
            loading={c.compare.isPending}
            onClick={c.readAgain}
            className="pointer-coarse:h-11"
          >
            {t('readAgain')}
          </Button>
        )}
      </div>
    </div>
  );
}

function ConnectForm({ c }: { c: Connect }) {
  const t = useT(connectMessages);
  const errorText = useErrorText();
  const busy = c.create.isPending;
  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    c.submit();
  };
  return (
    <form noValidate onSubmit={onSubmit} className="flex max-w-[720px] flex-col gap-8">
      <DestinationField
        value={c.values.panelId}
        disabled={busy}
        onChange={(panelId) => c.set({ panelId })}
      />
      <SourceFields values={c.values} set={c.set} disabled={busy} />
      {c.cert ? (
        <CertificateCard cert={c.cert} pending={busy} onTrust={() => c.cert && c.trust(c.cert)} />
      ) : null}
      {c.missing ? (
        <InlineAlert>{t(`missing_${c.missing}` as ConnectKey)}</InlineAlert>
      ) : c.create.error && !c.cert ? (
        <InlineAlert>{errorText(c.create.error)}</InlineAlert>
      ) : c.pin ? (
        <InlineAlert tone="info">{t('certTrusted')}</InlineAlert>
      ) : null}
      <div className="flex justify-end border-t border-kumo-hairline pt-5">
        <Button
          type="submit"
          variant="primary"
          loading={busy}
          className="w-full justify-center sm:w-auto pointer-coarse:h-11"
        >
          {t('connect')}
        </Button>
      </div>
    </form>
  );
}

/** Step 1: the destination, the old panel and its sign-in, then the live read. */
export function ConnectStep() {
  const tw = useT(wizardMessages);
  const t = useT(connectMessages);
  const c = useConnect();
  const failed = c.status === 'failed' || (c.status === 'idle' && c.compare.isError);
  return (
    <WizardFrame step="connect" description={tw('intro')}>
      {c.lost ? <InlineAlert tone="warning">{t('streamLost')}</InlineAlert> : null}
      {!c.jobId ? (
        <ConnectForm c={c} />
      ) : failed ? (
        <ReadStopped c={c} />
      ) : (
        <ReadProgress stream={c.stream} onStop={c.stop} stopping={c.cancel.isPending} />
      )}
    </WizardFrame>
  );
}
