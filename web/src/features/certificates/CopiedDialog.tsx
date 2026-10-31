import { Button } from '@cloudflare/kumo/components/button';
import { dialogMessages } from '../../components/dialogs/messages';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import type { CertCopy } from '../../lib/api/certificates';
import { useLingering } from '../../lib/motion/use-lingering';
import { BatchDialog } from '../batches/BatchDialog';
import { FactList } from '../../components/layout/FactList';
import { copyMessages } from './copy-messages';
import { Mono } from './parts';

/** The files a copy wrote, and that whoever reads them must point to them. */
export function CopiedDialog({ copy, onClose }: { copy: CertCopy | null; onClose: () => void }) {
  const t = useT(copyMessages);
  const td = useT(dialogMessages);
  const shown = useLingering(copy);
  return (
    <BatchDialog
      open={copy !== null}
      onOpenChange={(o) => !o && onClose()}
      title={t('copiedTitle')}
      footer={
        <Button variant="primary" className="pointer-coarse:h-9" onClick={onClose}>
          {td('close')}
        </Button>
      }
    >
      <p className="text-kumo-subtle">{t('copiedBody')}</p>
      {shown ? (
        <>
          <FactList
            facts={[
              {
                label: t('certFile'),
                value: <Mono text={shown.certFile} className="select-all" />,
              },
              { label: t('keyFile'), value: <Mono text={shown.keyFile} className="select-all" /> },
            ]}
          />
          <InlineAlert tone="info">
            {t(shown.target === 'panel' ? 'panelNote' : 'liveNote')}
          </InlineAlert>
        </>
      ) : null}
    </BatchDialog>
  );
}
