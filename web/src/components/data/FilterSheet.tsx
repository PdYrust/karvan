import { Select } from '@cloudflare/kumo/components/select';
import { useEffect, useState } from 'react';
import { useT } from '../../i18n/locale';
import { SheetDialog } from '../dialogs/SheetDialog';
import { currentOf, OptionText, type ToolbarChoice } from './ChoiceMenu';
import { dataMessages } from './messages';
/**
 * A list toolbar's filters and sorts on a phone: a bottom sheet with a labelled select for
 * each. It mounts on the first press and opens a frame later, so it still slides up.
 */
export default function FilterSheet({
  choices,
  open,
  onOpenChange,
}: {
  choices: readonly ToolbarChoice[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT(dataMessages);
  const [mounted, setMounted] = useState(false);
  useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);
  return (
    <SheetDialog open={open && mounted} onOpenChange={onOpenChange} title={t('filter')}>
      {choices.map((c) => (
        <Select
          key={c.label}
          label={c.label}
          className="w-full"
          value={c.value}
          renderValue={() => currentOf(c)?.label}
          onValueChange={(v) => {
            if (typeof v === 'string') c.onChange(v);
          }}
        >
          {c.options.map((o) => (
            <Select.Option key={o.value} value={o.value}>
              <span className="flex w-full items-center gap-2">
                <OptionText option={o} />
              </span>
            </Select.Option>
          ))}
        </Select>
      ))}
    </SheetDialog>
  );
}
