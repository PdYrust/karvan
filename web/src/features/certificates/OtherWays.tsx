import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { CaretDownIcon } from '@phosphor-icons/react';
import { useState, type ComponentProps } from 'react';
import { SheetDialog } from '../../components/dialogs/SheetDialog';
import { useT } from '../../i18n/locale';
import type { Way } from './domain-form';
import { domainFormMessages } from './domain-messages';

function Trigger({ className, ...props }: ComponentProps<typeof Button>) {
  const t = useT(domainFormMessages);
  return (
    <Button {...props} className={`self-start pointer-coarse:h-9 ${className ?? ''}`}>
      {t('otherWays')}
      <CaretDownIcon size={14} aria-hidden="true" className="text-kumo-subtle" />
    </Button>
  );
}

/**
 * Every way that is not the chosen one, No certificate last after a separator:
 * a menu from 640 px, the bottom sheet on phones.
 */
export function OtherWays({ ways, onWay }: { ways: readonly Way[]; onWay: (w: Way) => void }) {
  const t = useT(domainFormMessages);
  const [sheet, setSheet] = useState(false);
  const main = ways.filter((w) => w !== 'none');
  const none = ways.includes('none');
  return (
    <>
      <DropdownMenu>
        <DropdownMenu.Trigger render={<Trigger className="max-sm:hidden" />} />
        <DropdownMenu.Content align="start" className="min-w-56">
          {main.map((w) => (
            <DropdownMenu.Item key={w} onClick={() => onWay(w)}>
              {t(`way_${w}`)}
            </DropdownMenu.Item>
          ))}
          {none ? (
            <>
              <DropdownMenu.Separator />
              <DropdownMenu.Item onClick={() => onWay('none')}>{t('way_none')}</DropdownMenu.Item>
            </>
          ) : null}
        </DropdownMenu.Content>
      </DropdownMenu>
      <Trigger className="sm:hidden" onClick={() => setSheet(true)} />
      <SheetDialog open={sheet} onOpenChange={setSheet} title={t('otherWays')}>
        <ul className="flex flex-col overflow-hidden divide-y divide-kumo-hairline rounded-lg bg-kumo-base ring ring-kumo-hairline">
          {ways.map((w) => (
            <li key={w}>
              <button
                type="button"
                className="flex min-h-11 w-full items-center px-4 py-2.5 text-start text-kumo-default hover:bg-kumo-tint focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-kumo-brand"
                onClick={() => {
                  setSheet(false);
                  onWay(w);
                }}
              >
                {t(`way_${w}`)}
              </button>
            </li>
          ))}
        </ul>
      </SheetDialog>
    </>
  );
}
