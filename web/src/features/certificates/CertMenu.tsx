import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { CheckIcon } from '@phosphor-icons/react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { CopyIcon, DotsThreeIcon, TrashIcon } from '../../components/icons/regular';
import { useSubToggle } from '../../components/menus/use-sub-toggle';
import { PanelLabel } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import {
  certificateKeys,
  copyCertificate,
  type CertCopy,
  type Certificate,
  type CopyTarget,
} from '../../lib/api/certificates';
import { usePanels } from '../panels/use-panels';
import { CopiedDialog } from './CopiedDialog';
import { copyChoices } from './copy-targets';
import { copyMessages } from './copy-messages';
import { useCopyErrorText } from './copy-words';
import { certificatesMessages } from './messages';

/** Nothing uses it, so it can go; Karvan refuses the delete otherwise. */
const isUnused = (c: Certificate) => c.domains.length === 0 && c.rules.length === 0;

/**
 * Copy to a panel as a submenu that opens under its row, like Renew now: the Let's Encrypt live
 * folder, then each panel by name; where it is copied already is checked. The panels load when
 * it opens.
 */
export function CopySub({
  cert,
  busy,
  onCopy,
}: {
  cert: Certificate;
  busy: boolean;
  onCopy: (t: CopyTarget) => void;
}) {
  const t = useT(copyMessages);
  const { root, trigger } = useSubToggle();
  const panels = usePanels(root.open);
  const rows = copyChoices(cert, panels.data ?? []);
  return (
    <DropdownMenu.Sub {...root}>
      <DropdownMenu.SubTrigger
        className="gap-2 rtl:[&>svg:last-child]:-scale-x-100"
        disabled={busy}
        {...trigger}
      >
        <CopyIcon size={16} />
        {t('copyToPanel')}
      </DropdownMenu.SubTrigger>
      <DropdownMenu.SubContent
        side="bottom"
        align="start"
        sideOffset={4}
        className="w-max max-w-[min(20rem,calc(100vw-2rem))] min-w-(--anchor-width)"
      >
        {rows.map((r) => (
          <DropdownMenu.Item key={r.key} disabled={busy} onClick={() => onCopy(r.target)}>
            <span className="flex w-full min-w-0 items-center gap-2">
              {r.panel ? (
                <PanelLabel kind={r.panel.kind} name={r.panel.name} className="truncate" />
              ) : (
                <span className="truncate">{t('letsEncryptLive')}</span>
              )}
              <span className="ms-auto flex w-4 shrink-0 justify-center">
                {r.copied ? <CheckIcon size={14} aria-hidden="true" /> : null}
              </span>
            </span>
          </DropdownMenu.Item>
        ))}
        {panels.isPending && root.open ? (
          <DropdownMenu.Item disabled>{t('loadingPanels')}</DropdownMenu.Item>
        ) : null}
      </DropdownMenu.SubContent>
    </DropdownMenu.Sub>
  );
}

/** A certificate's "..." menu: copy it to a panel, or delete it while nothing uses it. */
export function CertMenu({ cert, onDelete }: { cert: Certificate; onDelete: () => void }) {
  const t = useT(certificatesMessages);
  const tc = useT(copyMessages);
  const copyError = useCopyErrorText();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  const [copied, setCopied] = useState<CertCopy | null>(null);
  const label = t('moreFor', { name: cert.name });
  const copy = useMutation({
    mutationFn: (target: CopyTarget) => copyCertificate(cert.id, target),
    onSuccess: (c) => {
      setCopied(c);
      void qc.invalidateQueries({ queryKey: certificateKeys.all });
    },
    onError: (e) =>
      toasts.add({ title: tc('copyFailed'), description: copyError(e), variant: 'error' }),
  });
  return (
    <>
      <DropdownMenu>
        <DropdownMenu.Trigger
          render={
            <Button
              variant="ghost"
              shape="square"
              aria-label={label}
              title={label}
              icon={<DotsThreeIcon size={20} />}
              className="shrink-0 pointer-coarse:size-9"
            />
          }
        />
        <DropdownMenu.Content align="end" className="min-w-52">
          <CopySub cert={cert} busy={copy.isPending} onCopy={(target) => copy.mutate(target)} />
          <DropdownMenu.Item
            className="gap-2"
            variant="danger"
            disabled={!isUnused(cert)}
            onClick={onDelete}
          >
            <TrashIcon size={16} />
            {t('delete')}
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu>
      <CopiedDialog copy={copied} onClose={() => setCopied(null)} />
    </>
  );
}
