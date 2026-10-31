import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { PanelLabel } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import {
  certificateKeys,
  stopCertificateCopy,
  type CertCopy,
  type Certificate,
} from '../../lib/api/certificates';
import { useErrorText } from '../../lib/errors/errors';
import { copyMessages } from './copy-messages';
import { useCopyReason } from './copy-words';
import { certificatesMessages } from './messages';
import { Mono } from './parts';

/** One folder: whose it is, its path, a failed last copy, and Stop copying. */
function Target({ copy, onStop, busy }: { copy: CertCopy; onStop: () => void; busy: boolean }) {
  const t = useT(copyMessages);
  const reason = useCopyReason();
  return (
    <li className="flex min-w-0 flex-col items-start gap-0.5">
      <span className="flex max-w-full min-w-0 flex-wrap items-center gap-x-2">
        {copy.panel ? (
          <PanelLabel
            kind={copy.panel.kind || undefined}
            name={copy.panel.name}
            className="truncate"
          />
        ) : (
          <span className="truncate">{t('letsEncryptLive')}</span>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="pointer-coarse:h-9"
          aria-label={t('stopCopyFor', { dir: copy.dir })}
          disabled={busy}
          onClick={onStop}
        >
          {t('stopCopy')}
        </Button>
      </span>
      <Mono text={copy.dir} className="text-sm text-kumo-subtle" />
      {copy.error ? (
        <span className="text-sm">
          <StatusLabel tone="warning">
            {t('lastFailed', { reason: reason(copy.error) })}
          </StatusLabel>
        </span>
      ) : null}
    </li>
  );
}

/** The folders Karvan keeps a certificate copied to, each with Stop copying; the files stay. */
export function CopyTargets({ cert }: { cert: Certificate }) {
  const t = useT(copyMessages);
  const tc = useT(certificatesMessages);
  const errorText = useErrorText();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  const stop = useMutation({
    mutationFn: (c: CertCopy) => stopCertificateCopy(cert.id, c.id),
    onSuccess: () => {
      toasts.add({ title: t('copyStopped') });
      void qc.invalidateQueries({ queryKey: certificateKeys.all });
    },
    onError: (e) =>
      toasts.add({ title: t('stopFailed'), description: errorText(e), variant: 'error' }),
  });
  if (cert.copies.length === 0) return <span className="text-kumo-subtle">{tc('none')}</span>;
  return (
    <ul className="flex min-w-0 flex-col gap-2">
      {cert.copies.map((c) => (
        <Target key={c.id} copy={c} busy={stop.isPending} onStop={() => stop.mutate(c)} />
      ))}
    </ul>
  );
}
