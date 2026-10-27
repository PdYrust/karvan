import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { XIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';
import { builtPages } from './routes-built';

const isMac = /Mac|iPhone|iPad/.test(navigator.platform);
const kbd =
  'inline-grid h-6 min-w-6 place-items-center rounded-md bg-kumo-base px-1.5 font-mono text-tiny text-kumo-default ring ring-kumo-line';

function Keys({ keys, then }: { keys: string[]; then?: string }) {
  return (
    <span dir="ltr" className="flex items-center gap-1">
      {keys.map((k, i) => (
        <span key={k} className="flex items-center gap-1">
          {i > 0 && then ? <span className="text-tiny text-kumo-subtle">{then}</span> : null}
          <kbd className={kbd}>{k}</kbd>
        </span>
      ))}
    </span>
  );
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <li className="flex min-h-10 items-center justify-between gap-4 border-b border-kumo-hairline last:border-b-0">
      <span>{label}</span>
      {children}
    </li>
  );
}

/** Every keyboard shortcut in one list (`?`). */
export function ShortcutSheet({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const t = useT(shellMessages);
  const then = t('shortcutsThen');
  const letter = (code: string | undefined) => (code ?? '').replace('Key', '').toLowerCase();
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog size="lg" className="p-6">
        <div className="mb-4 flex items-start justify-between gap-4">
          <Dialog.Title className="text-heading font-semibold">{t('shortcuts')}</Dialog.Title>
          <Dialog.Close
            render={(props) => (
              <Button
                {...props}
                variant="ghost"
                shape="square"
                className="pointer-coarse:size-9"
                aria-label={t('close')}
                icon={<XIcon size={18} />}
              />
            )}
          />
        </div>
        <ul className="flex flex-col">
          <Row label={t('shortcutsSearch')}>
            <Keys keys={[isMac ? '⌘' : 'Ctrl', 'K']} />
          </Row>
          <Row label={t('shortcutsHelp')}>
            <Keys keys={['?']} />
          </Row>
          <Row label={t('shortcutsPageSearch')}>
            <Keys keys={['/']} />
          </Row>
          {builtPages().map((p) => (
            <Row key={p.label} label={t('shortcutsGoto', { page: t(p.label) })}>
              <Keys keys={['g', letter(p.key)]} then={then} />
            </Row>
          ))}
          <Row label={t('shortcutsGoto', { page: t('settings') })}>
            <Keys keys={['g', 's']} then={then} />
          </Row>
        </ul>
      </Dialog>
    </Dialog.Root>
  );
}
